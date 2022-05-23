package work

import (
	"context"
	"fmt"
	lapi "github.com/filecoin-project/lotus/api"
	"github.com/ipfs/go-cidutil/cidenc"
	"github.com/multiformats/go-multibase"
	"io/ioutil"
	"log"
	"order/lotuss"
	"path"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
)

//lotus client deal --fast-retrieval=true --from=$clientwallet --manual-piece-cid=$PiecesCID --manual-piece-size=1065353216 --manual-stateless-deal --verified-deal=false --start-epoch=1299917 $fileCID $MinerAddress 0 1468800
//--fast-retrieval=true  是否支持快速检索
//--from= 发单方扣币地址
//--manual-piece-cid=   “lotus client commP”返回的CID
//--manual-piece-size= 单位为字节。具体数字可参考：根据第3不算出的Piece size结果进行2的n次方对齐运算。例如1.823MiB，对齐后就是2M，然后根据下表算出Usable size，结果就是manual-piece-size
//--manual-stateless-deal 表示此订单为离线交易
//--verified-deal=false 表示此订单非已验证订单
//--start-epoch= 指定交易最晚完成时间，一般设置为发单时高度+7天 （当前高度+20160）
//$fileCID   “lotus client import”返回的CID
//$MinerAddress：StorageProvider节点号
//0 单价
//1468800 存储期限

var (
	carpool chan map[string]bool
	rres    = make(map[string]string)
)

type ImportCar struct {
	CarPath string `json:"carPath"`
	Import  int64  `json:"import"`
	Root    string `json:"root"`
}
type CommPCar struct {
	CarPath string `json:"carPath"`
	CID     string `json:"cid"`
	Size    string `json:"size"`
}

type RES struct {
	Import int64  `json:"import"`
	Root   string `json:"rootCid"`
	CID    string `json:"manual-piece-cid"`
	Size   string `json:"manual-piece-size"`
	Epoch  int64  `json:"start-epoch"`
}

func DoWork(ctx context.Context, inputPath, outPath, minerID string) {
	var carmap map[string]bool
	fil, err := GetAllFile(inputPath)
	if err != nil {
		log.Fatalf("GetAllFile  panic: %v, stack: %s", err, debug.Stack())
		return
	}

	for _, s := range fil {
		absPath, err := filepath.Abs(s)
		if err != nil {
			log.Fatalf("get file abs is failed:%v", err)
			return
		}
		filesuffix := path.Ext(s)
		filenameall := path.Base(s)
		name := strings.TrimSuffix(filenameall, filesuffix)
		ref := lapi.FileRef{
			Path:  absPath,
			IsCAR: strings.Contains(s, ".car"),
		}
		out_ := outPath + "/" + name + ".car"
		err = lotuss.Node().ClientGenCar(ctx, ref, out_)
		if err != nil {
			fmt.Println(err)
			return
		}
		carmap[out_] = false
	}
	carpool <- carmap

	for {
		select {
		case <-ctx.Done():
			break
		case _, ok := <-carpool:
			if !ok {
				log.Printf("read carpool  is not ok")
			}

		}

	}
}

func DoWorkCar(ctx context.Context, path string, minerID string) {

	err, i := ClientImport(ctx, path)
	if err != nil {
		log.Println("ClientImport is err:", err)
	}
	err, c := ClientCommP(ctx, path)
	if err != nil {
		log.Println("ClientCommP is err:", err)
	}
	for _, car := range i {
		var r string
		for _, pCar := range c {
			r = fmt.Sprintf("lotus client deal --fast-retrieval=true --manual-piece-cid=%s  --manual-piece-size=%s --manual-stateless-deal --verified-deal=false  --start-epoch=%d %s %s 0 1468800", pCar.CID, pCar.Size, 7*2880+1832824, car.Root, minerID)
		}
		rres[car.CarPath] = r
	}
	fmt.Println(rres)

}

func ClientImport(ctx context.Context, path string) (error, []ImportCar) {
	defer func() {
		if err := recover(); err != nil {
			log.Fatalf("ClientImport  panic: %v, stack: %s", err, debug.Stack())
		}
	}()
	var cars []ImportCar
	pathname, err := GetAllFile(path)
	if err != nil {
		log.Fatalf("get file is failed:%v", err)
	}
	for _, s := range pathname {
		var car ImportCar
		absPath, err := filepath.Abs(s)
		if err != nil {
			log.Fatalf("get file abs is failed:%v", err)

			return err, cars
		}
		ref := lapi.FileRef{
			Path:  absPath,
			IsCAR: strings.Contains(s, ".car"),
		}
		c, err := lotuss.Node().ClientImport(ctx, ref)
		if err != nil {
			return err, cars
		}
		encoder, err := GetCidEncoder(ctx)
		if err != nil {
			log.Fatalf("encoder file is failed:%v", err)
			return err, cars
		}
		car.CarPath = absPath
		car.Import = int64(c.ImportID)
		car.Root = encoder.Encode(c.Root)
		cars = append(cars, car)
	}
	return nil, cars
}

func ClientCommP(ctx context.Context, path string) (error, []CommPCar) {
	defer func() {
		if err := recover(); err != nil {
			log.Fatalf("ClientCommP  panic: %v, stack: %s", err, debug.Stack())
		}
	}()
	var CommPs []CommPCar
	pathname, err := GetAllFile(path)
	if err != nil {
		log.Fatalf("get file is failed:%v", err)
		return nil, CommPs
	}
	for _, s := range pathname {
		var ccar CommPCar

		if !strings.Contains(s, ".car") {
			continue
		}
		ret, err := lotuss.Node().ClientCalcCommP(ctx, s)
		if err != nil {
			return err, CommPs
		}
		encoder, err := GetCidEncoder(ctx)
		if err != nil {
			log.Fatalf("encoder file is failed:%v", err)
			return err, CommPs
		}
		fmt.Println(ret.Size)
		ccar.CID = encoder.Encode(ret.Root)
		ccar.Size = strconv.FormatUint(uint64(ret.Size), 10)
		CommPs = append(CommPs, ccar)
	}

	return nil, CommPs
}

// GetAllFile 递归获取指定目录下的所有文件名
func GetAllFile(pathname string) ([]string, error) {
	var result []string
	fis, err := ioutil.ReadDir(pathname)
	if err != nil {
		fmt.Printf("读取文件目录失败，pathname=%v, err=%v \n", pathname, err)

		return result, err
	}

	// 所有文件/文件夹
	for _, fi := range fis {
		filename := pathname + "/" + fi.Name()
		if fi.IsDir() {
			temp, err := GetAllFile(filename)
			if err != nil {
				fmt.Printf("读取文件目录失败,filename=%v, err=%v", filename, err)
				return result, err
			}
			result = append(result, temp...)
		} else {
			result = append(result, filename)
		}
	}

	return result, nil
}

// GetCidEncoder returns an encoder using the `cid-base` flag if provided, or
// the default (Base32) encoder if not.

func GetCidEncoder(ctx context.Context) (cidenc.Encoder, error) {
	e := cidenc.Encoder{Base: multibase.MustNewEncoder(multibase.Base32)}

	return e, nil
}
