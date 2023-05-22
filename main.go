package main

import (
	"fmt"
	"os"
	"time"

	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/library-go/jsonhelper"
)

type (
	userpw struct {
		UserName     string                  `json:"username"`
		Password     string                  `json:"password"`
		RefreshToken string                  `json:"refreshToken"`
		WebToken     aliyunpan.WebLoginToken `json:"webToken"`
	}
)

var totalFile int64 = 0

func main() {
	panClient, ui := doLogin()
	finfo, errFinfo := panClient.FileInfoByPath(ui.FileDriveId, "/export008")
	fmt.Println(finfo, errFinfo)
	listparam := &aliyunpan.FileListParam{}
	listparam.DriveId = ui.FileDriveId
	listparam.ParentFileId = finfo.FileId
	fl, errListGetAll := panClient.FileListGetAll(listparam, 1000)
	if nil != errListGetAll {
		fmt.Println(fl, errListGetAll)
		return
	}
	dataFile, _ := os.OpenFile("data_"+finfo.FileName+".json", os.O_CREATE|os.O_RDWR, 0755)
	numFile, numDir := fl.Count()
	totalFile += numFile
	fmt.Fprintln(os.Stdout, "totalFile:\t\t", totalFile)
	for i := 0; i < int(numDir+numFile); i++ {
		file := fl.Item(i)
		fmt.Println(file.FileName)
		jsonhelper.MarshalData(dataFile, file)
		if file.FileType == "folder" {
			time.Sleep(1 * time.Second)
			ListSubDir(panClient, ui, file, dataFile)
		}
	}
}
func ListSubDir(panclient *aliyunpan.PanClient, userinfo *aliyunpan.UserInfo, pfile *aliyunpan.FileEntity, datafile *os.File) {
	listparam := &aliyunpan.FileListParam{}
	listparam.DriveId = userinfo.FileDriveId
	listparam.ParentFileId = pfile.FileId
	fl, errListGetAll := panclient.FileListGetAll(listparam, 1200)
	if nil != errListGetAll {
		fmt.Println(fl, errListGetAll)
	}
	numFile, numDir := fl.Count()
	totalFile += numFile
	fmt.Fprintln(os.Stdout, "totalFile:\t\t", totalFile)
	for i := 0; i < int(numDir+numFile); i++ {
		file := fl.Item(i)
		fmt.Println(file.FileName)
		jsonhelper.MarshalData(datafile, file)
		if file.FileType == "folder" {
			time.Sleep(1 * time.Second)
			ListSubDir(panclient, userinfo, file, datafile)
			// mvSubFolderItems2Parent(panclient, userinfo, file, pfile)
		}
	}
}
func mvSubFolderItems2Parent(panclient *aliyunpan.PanClient, userinfo *aliyunpan.UserInfo, pfile *aliyunpan.FileEntity, ppfile *aliyunpan.FileEntity) {
	// mv := &aliyunpan.FileMoveParam{}
	// panclient.FileMove(mv)
	listparam := &aliyunpan.FileListParam{}
	listparam.DriveId = userinfo.FileDriveId
	listparam.ParentFileId = pfile.FileId
	fl, errListGetAll := panclient.FileListGetAll(listparam, 1200)
	if nil != errListGetAll {
		fmt.Println(fl, errListGetAll)
		panclient, userinfo = doLogin()
	}
	numFile, numDir := fl.Count()
	totalFile += numFile
	fmt.Fprintln(os.Stdout, "totalFile:\t\t", totalFile)
	var filearr [](*aliyunpan.FileMoveParam)
	for i := 0; i < int(numDir+numFile); i++ {
		file := fl.Item(i)
		fmt.Print(file.FileName)
		tmpMv := &aliyunpan.FileMoveParam{}
		tmpMv.DriveId = userinfo.FileDriveId
		tmpMv.FileId = file.FileId
		tmpMv.ToDriveId = userinfo.FileDriveId
		tmpMv.ToParentFileId = pfile.ParentFileId
		filearr = append(filearr, tmpMv)
		fmt.Printf("filearr: %v\n", filearr)
		time.Sleep(1 * time.Second)
		panclient.FileMove(filearr)
	}
}
func doLogin() (*aliyunpan.PanClient, *aliyunpan.UserInfo) {
	configFile, err := os.OpenFile("config.json", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		fmt.Println("read user info error")
		return nil, nil
	}
	defer configFile.Close()

	localconfig := &userpw{}
	err = jsonhelper.UnmarshalData(configFile, localconfig)
	if err != nil {
		fmt.Println("read user info error")
		return nil, nil
	}
	fmt.Println(localconfig.WebToken)
	webToken := &aliyunpan.WebLoginToken{}
	webToken = &localconfig.WebToken
	if localconfig.WebToken.AccessToken == "" {
		tmpWebToken, err := aliyunpan.GetAccessTokenFromRefreshToken(localconfig.RefreshToken)
		if err != nil {
			fmt.Println("get acccess token error")
			return nil, nil
		}
		localconfig.WebToken = *tmpWebToken
		jsonhelper.MarshalData(configFile, localconfig)
	}

	appConfig := aliyunpan.AppConfig{
		AppId:     "25dzX3vbYqktVxyX",
		DeviceId:  "WnIrdZj0pnOZpaR0LOnrsI5O",
		UserId:    "b88f5ca8f4d2454b80e1e3c419b95f7d",
		Nonce:     0,
		PublicKey: "",
	}
	// pan client
	panClient := aliyunpan.NewPanClient(*webToken, aliyunpan.AppLoginToken{}, appConfig, aliyunpan.SessionConfig{
		DeviceName: "Chrome浏览器",
		ModelName:  "Windows网页版",
	})
	r, e := panClient.CreateSession(nil)
	if e != nil {
		fmt.Println("call CreateSession error in SetupUserByCookie: " + e.Error())
	}
	if r != nil && !r.Result {
		fmt.Println("上传签名秘钥失败，可能是你账号登录的设备已超最大数量")
	}
	// get user info
	ui, errUserInfo := panClient.GetUserInfo()
	if nil != errUserInfo {
		fmt.Println("get user info error")
		fmt.Println(ui, errUserInfo)
		return nil, nil
	}
	return panClient, ui
}
