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
	"sync"
)

var wg sync.WaitGroup // 等待组

func DoWork(ctx context.Context, inputPath, outPath string) {
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
	}
}

func DoWorkCar(ctx context.Context, path string) {
	wg.Add(2)
	go func() {
		err, _ := ClientImport(ctx, path)
		if err != nil {
			log.Println("ClientImport is err:", err)
		}
	}()
	go func() {
		err, _ := ClientCommP(ctx, path)
		if err != nil {
			log.Println("ClientCommP is err:", err)

		}
	}()
	wg.Wait()

}

func ClientImport(ctx context.Context, path string) (error, string) {
	defer func() {
		if err := recover(); err != nil {
			log.Fatalf("ClientImport  panic: %v, stack: %s", err, debug.Stack())
		}
	}()
	pathname, err := GetAllFile(path)
	if err != nil {
		log.Fatalf("get file is failed:%v", err)
	}
	for _, s := range pathname {
		absPath, err := filepath.Abs(s)
		if err != nil {
			log.Fatalf("get file abs is failed:%v", err)

			return err, ""
		}

		ref := lapi.FileRef{
			Path:  absPath,
			IsCAR: strings.Contains(s, ".car"),
		}
		c, err := lotuss.Node().ClientImport(ctx, ref)
		if err != nil {
			return err, ""
		}
		encoder, err := GetCidEncoder()
		if err != nil {
			log.Fatalf("encoder file is failed:%v", err)
			return err, ""
		}
		fmt.Println(encoder.Encode(c.Root))
	}
	wg.Done()

	return nil, ""
}

func ClientCommP(ctx context.Context, path string) (error, string) {
	defer func() {
		if err := recover(); err != nil {
			log.Fatalf("ClientCommP  panic: %v, stack: %s", err, debug.Stack())
		}
	}()
	pathname, err := GetAllFile(path)
	if err != nil {
		log.Fatalf("get file is failed:%v", err)
		return nil, ""
	}
	for _, s := range pathname {
		if !strings.Contains(s, ".car") {
			continue
		}
		ret, err := lotuss.Node().ClientCalcCommP(ctx, s)
		if err != nil {
			return err, ""
		}
		encoder, err := GetCidEncoder()
		if err != nil {
			log.Fatalf("encoder file is failed:%v", err)
			return err, ""
		}

		fmt.Println("CID: ", encoder.Encode(ret.Root))
		fmt.Println("Piece size: ", types.SizeStr(types.NewInt(uint64(ret.Size))))
	}
	wg.Done()

	return nil, ""
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

func GetCidEncoder() (cidenc.Encoder, error) {
	e := cidenc.Encoder{Base: multibase.MustNewEncoder(multibase.Base32)}
	return e, nil
}
