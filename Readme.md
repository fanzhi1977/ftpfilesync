#  Ftpfilesync基于ftp协议的文件同步程序
## 基于github.com/jlaffaye/ftp组件，使用go语言编写
## 配置指南
配置文件名为config.json，需放置到可执行文件相同的路径。
```json
{
  "Debug": true,
  "Host": "192.168.100.110",
  "Port": "21",
  "User": "test",
  "Passwd": "test",
  "Transfers":[
    {
      "FtpDir": "send/",
      "LocalDir": "/Users/fanzhi/Documents/temp/send/",
      "IsPut": true
    },
    {
      "FtpDir": "receive/",
      "LocalDir": "/Users/fanzhi/Documents/temp/receive/",
      "IsPut": false
    }
  ],
  "DeleteSource": true,
  "Sleep": 100,
  "Filefilters": [".dat",".abc"]
}
```
* Debug bool是否为debug模式
* Host ftp服务器账号
* Port ftp服务器的端口
* User ftp服务器的用户名
* Passwd ftp服务器的密码
* Transfers 传输配置
    * FtpDir ftp服务器的文件相对路径
    * LocalDir 本地文件夹路径
    * IsPut 是上传还是下载，true即上传
* DeleteSource 是否删除源
* Sleep 休眠时间，毫秒
* Filefilters 后缀名过滤