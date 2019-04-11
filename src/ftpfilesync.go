package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jlaffaye/ftp"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"
)

var config Config
var waitgroup sync.WaitGroup // 记录进程处理结束


func main() {
	flaConfigPath := "config.json"
	if !checkFileIsExist(flaConfigPath) {
		fmt.Println("当前目录缺少文件: ", flaConfigPath)
		return
	}
	readConfig(flaConfigPath)
	//拔号
	c, err := ftp.DialWithOptions(config.Host+":"+config.Port, ftp.DialWithTimeout(5*time.Second))

	if err != nil {
		fmt.Println(err)
	}
	//登录
	err = c.Login(config.User, config.Passwd)
	if err != nil {
		fmt.Println(err)
	}

	//创建目标文件夹
	for _, t := range config.Transfers {
		remoteDir := t.FtpDir
		localDir := t.LocalDir
		IsPut := t.IsPut
		if IsPut {
			//上传时在ftp服务器创建文件夹
			c.MakeDir(remoteDir)
		} else {
			//下载时在本地创建文件夹
			os.Mkdir(localDir,0666)
		}
	}
	//执行操作
	for true {
		for _, t := range config.Transfers {
			remoteDir := t.FtpDir
			localDir := t.LocalDir
			IsPut := t.IsPut


			if IsPut {
				syncUpload(localDir, remoteDir,c)
			} else {
				//下载
				syncDownload(localDir, remoteDir,c)
			}
		}
		time.Sleep(time.Duration(config.Sleep) * time.Millisecond)
	}

	// 退出登录
	if err := c.Quit(); err != nil {
		//t.Fatal(err)
		fmt.Println(err)
	}
}

//同步，上传
func syncUpload( localDir string, remoteDir string,conn* ftp.ServerConn){
	//上传文件
	localDirs, _ := ioutil.ReadDir(localDir)
	fmt.Println(time.Now().Format("2006-01-02 15:04:05.000"),":开始上传文件,",localDir,",文件数量为：", len(localDirs))
	for i, localResource := range localDirs {
		if !localResource.IsDir() {
			//文件时处理
			fileName := localResource.Name()
			if !filterMatch(fileName,config.Filefilters){
				//后缀不匹配时直接跳过
				continue
			}
			ftpFileName := remoteDir + fileName
			localFileName := localDir + fileName
			byteFile, e1 := ioutil.ReadFile(localFileName)
			if e1 != nil {
				fmt.Println(e1)
				continue
			}
			conn.Stor(ftpFileName,bytes.NewBuffer(byteFile))
			if config.DeleteSource{
				//删除原文件
				e:=os.Remove(localFileName)
				if e != nil {
					fmt.Println(e)
					continue
				}

			}
			fmt.Println(time.Now().Format("2006-01-02 15:04:05.000"),":上传文件,序号：",i,",文件名：",fileName)
		}
	}
	waitgroup.Wait() //Wait()这里会发生阻塞，直到队列中所有的任务结束就会解除阻塞
}
//同步，下载
func syncDownload( localDir string, remoteDir string,conn* ftp.ServerConn){
	//上传文件
	entries,err := conn.List(remoteDir) //列出无端目录

	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(time.Now().Format("2006-01-02 15:04:05.000"),":开始下载文件,",remoteDir,",文件数量为：", len(entries))

	for i,entry :=range entries{
		//遍历ftp端目录
		if entry.Type==ftp.EntryTypeFile{
			//文件时处理，只同步文件
			fileName:=entry.Name
			if !filterMatch(fileName,config.Filefilters){
				//后缀不匹配时直接跳过
				continue
			}
			ftpFileName:=remoteDir+fileName
			localFileName := localDir + fileName
			r,e:=conn.Retr(ftpFileName)
			if e != nil {
				fmt.Println(e)
				continue
			}
			bytes,e1:=ioutil.ReadAll(r)
			if e1 != nil {
				fmt.Println(e1)
				continue
			}
			e2:=ioutil.WriteFile(localFileName,bytes,0666)
			if e2 != nil {
				fmt.Println(e2)
				continue
			}
			e3:=r.Close()
			if e3 != nil {
				fmt.Println(e3)
				continue
			}
			if config.DeleteSource{
				//删除原文件
				e4:=conn.Delete(ftpFileName)
				if e4 != nil {
					fmt.Println(e4)
					continue
				}
			}

			fmt.Println(time.Now().Format("2006-01-02 15:04:05.000"),":下载文件,序号：",i,",文件名：",fileName)
		}
	}
	waitgroup.Wait() //Wait()这里会发生阻塞，直到队列中所有的任务结束就会解除阻塞
}


type Config struct {
	Host         string     // ip
	Port         string     // 端口
	User         string     // 用户名
	Passwd       string     // 密码
	Transfers    []Transfer //定义上传还是下载
	CpuDouble    int        // cpu倍数
	Debug        bool
	DeleteSource bool
	Sleep        int
	Filefilters  []string
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
	bytes, _ := ioutil.ReadFile(path)
	err := json.Unmarshal(bytes, &config)
	if err != nil {
		fmt.Println("error in translating,", err.Error())
	}
}
//过滤文件名
func filterMatch(name string,matcher []string) bool{
	for _,s :=range matcher{
		if strings.HasSuffix(name,s){
			//匹配时直接返回true
			return true
		}
	}
	return false
}
