package main

import (
	"bytes"
	"encoding/json"
	"github.com/jlaffaye/ftp"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

var config Config

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
}

func main() {

	flaConfigPath := "config.json"
	if !checkFileIsExist(flaConfigPath) {
		log.Error("当前目录缺少文件: ", flaConfigPath)
		return
	}
	readConfig(flaConfigPath)
	if config.Debug {
		log.SetLevel(log.DebugLevel)
	}
	for {
		run()
	}
}

func run() {
	//拔号
	conn, err := ftp.DialWithOptions(config.Host+":"+config.Port, ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		log.Fatal(err)
		return
	}
	// 退出登录
	funcQuit := func() {
		if conn != nil {
			//连接未断开时退出
			e := conn.Quit()
			log.Fatal("出现错误，ftp连接断开 ")
			if e != nil {
				log.Error(e)
				return
			}
		}
	}
	defer funcQuit()

	//登录
	err = conn.Login(config.User, config.Passwd)
	if err != nil {
		log.Fatal(err)
		return
	}

	//创建目标文件夹
	for _, t := range config.Transfers {
		remoteDir := t.FtpDir
		localDir := t.LocalDir
		IsPut := t.IsPut
		if IsPut {
			//上传时在ftp服务器创建文件夹
			e := conn.MakeDir(remoteDir)
			if e != nil {
				log.Error(e)
			}
		} else {
			//下载时在本地创建文件夹
			_, e := os.Stat(localDir)
			if e != nil {
				//先检查文件夹是否存在
				e := os.Mkdir(localDir, 0666)
				if e != nil {
					log.Error(e)
				}
			}
		}
	}
	//执行任务
	doTask(conn)
	return
}

//执行操作
func doTask(conn *ftp.ServerConn) {
	for true {
		for _, t := range config.Transfers {
			remoteDir := t.FtpDir
			localDir := t.LocalDir
			IsPut := t.IsPut
			//检查服务器是否连接
			e := conn.NoOp()
			if e != nil {
				log.Fatal(e)
				return
			}
			if IsPut {
				syncUpload(localDir, remoteDir, conn)
			} else {
				//下载1
				syncDownload(localDir, remoteDir, conn)
			}
		}
		time.Sleep(time.Duration(config.Sleep) * time.Millisecond)
	}
}

//同步，上传
func syncUpload(localDir string, remoteDir string, conn *ftp.ServerConn) {
	//上传文件
	localDirs, _ := ioutil.ReadDir(localDir)
	log.Info("开始上传文件,", localDir, ",文件数量为：", len(localDirs))
	//开始执行时间
	start := time.Now().UnixNano() / 1e6
	j := 0
	for i, localResource := range localDirs {
		if !localResource.IsDir() {
			//文件时处理
			fileName := localResource.Name()
			if !filterMatch(fileName, config.Filefilters) {
				//后缀不匹配时直接跳过
				continue
			}
			ftpFileName := remoteDir + fileName
			localFileName := localDir + fileName
			byteFile, e1 := ioutil.ReadFile(localFileName)
			if e1 != nil {
				log.Error(e1)
				continue
			}
			e2 := conn.Stor(ftpFileName, bytes.NewBuffer(byteFile))
			if e2 != nil {
				log.Error(e2)
				continue
			}
			if config.DeleteSource {
				//删除原文件
				e := os.Remove(localFileName)
				if e != nil {
					log.Error(e)
					continue
				}

			}
			log.Debug("上传文件,序号：", i, ",文件名：", fileName)
			j = j + 1
		}
	}
	log.Info("上传文件共：", j, "个，耗时:", time.Now().UnixNano()/1e6-start, "毫秒")

}

//同步，下载
func syncDownload(localDir string, remoteDir string, conn *ftp.ServerConn) {
	//上传文件
	entries, err := conn.List(remoteDir) //列出无端目录

	if err != nil {
		log.Error(err)
		return
	}

	log.Info("开始下载文件,", remoteDir, ",文件数量为：", len(entries))
	start := time.Now().UnixNano() / 1e6
	j := 0
	for i, entry := range entries {
		//遍历ftp端目录
		if entry.Type == ftp.EntryTypeFile {
			//文件时处理，只同步文件
			fileName := entry.Name
			if !filterMatch(fileName, config.Filefilters) {
				//后缀不匹配时直接跳过
				continue
			}
			ftpFileName := remoteDir + fileName
			localFileName := localDir + fileName
			r, e := conn.Retr(ftpFileName)
			if e != nil {
				log.Error(e)
				continue
			}
			b, e1 := ioutil.ReadAll(r)
			if e1 != nil {
				log.Error(e1)
				continue
			}
			e2 := ioutil.WriteFile(localFileName, b, 0666)
			if e2 != nil {
				log.Error(e2)
				continue
			}
			e3 := r.Close()
			if e3 != nil {
				log.Error(e3)
				continue
			}
			if config.DeleteSource {
				//删除原文件
				e4 := conn.Delete(ftpFileName)
				if e4 != nil {
					log.Error(e4)

					continue
				}
			}
			log.Debug("下载文件,序号：", i, ",文件名：", fileName)
		}
		j++
	}
	log.Info("上传文件共：", j, "个，耗时:", time.Now().UnixNano()/1e6-start, "毫秒")

}

type Config struct {
	Host         string     // ip
	Port         string     // 端口
	User         string     // 用户名
	Passwd       string     // 密码
	Transfers    []Transfer //定义上传还是下载
	CpuDouble    int        // cpu倍数
	Debug        bool       //是否调度模式
	DeleteSource bool       //删除源文件
	Sleep        int        //休眠时长
	Filefilters  []string   //文件后缀过滤
	log          string     //日志目录
}

type Transfer struct {
	LocalDir  string // 本地位置
	LocalFile string // 本地文件
	FtpDir    string // ftp位置
	IsPut     bool   //是否上传
}

/**
* 判断文件是否存在 存在返回 true 不存在返回false
 */
func checkFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

// 读取配置
func readConfig(path string) {
	b, _ := ioutil.ReadFile(path)
	err := json.Unmarshal(b, &config)
	if err != nil {
		log.Error(err)
	}
}

//过滤文件名
func filterMatch(name string, matcher []string) bool {
	for _, s := range matcher {
		if strings.HasSuffix(name, s) {
			//匹配时直接返回true
			return true
		}
	}
	return false
}
