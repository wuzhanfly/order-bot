package work

import (
	"context"
	"fmt"
	lapi "github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cidutil/cidenc"
	"github.com/multiformats/go-multibase"
	"io/ioutil"
	"log"
	"order/lotuss"
	"path"
	"path/filepath"
	"runtime/debug"
	"strings"
)

var (
	carpool chan map[string]bool
	rres    map[string]*RES
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
	CarPath    string `json:"carPath"`
	OriginPath string `json:"originPath"`
	Import     int64  `json:"import"`
	Root       string `json:"rootCid"`
	CID        string `json:"cid"`
	Size       string `json:"size"`
}

func DoWork(ctx context.Context, inputPath, outPath string) {
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

func DoWorkCar(ctx context.Context, path string) {
	err, i := ClientImport(ctx, path)
	if err != nil {
		log.Println("ClientImport is err:", err)
	}

	err, c := ClientCommP(ctx, path)
	if err != nil {
		log.Println("ClientCommP is err:", err)

	}
	fmt.Println(i, c)
	for _, car := range i {
		var r RES
		for _, pCar := range c {
			r.OriginPath = car.CarPath
			r.CarPath = car.CarPath
			r.Root = car.Root
			r.Size = pCar.Size
			r.CID = pCar.CID
		}
		fmt.Println(car.CarPath, &r)
		rres[car.CarPath] = &r
	}
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
		ccar.CID = encoder.Encode(ret.Root)
		ccar.Size = types.SizeStr(types.NewInt(uint64(ret.Size)))
		CommPs = append(CommPs, ccar)
	}
	//for _, p := range CommPs {
	//	if _, ok := res[p.CarPath]; ok {
	//		res[p.Size] = p.Size
	//		res[p.CID] = p.CID
	//	}
	//}

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
		// 是文件夹则递归进入获取;是文件，则压入数组
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
