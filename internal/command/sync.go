// Copyright (c) 2020 tickstep.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package command

import (
	"fmt"
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/syncdrive"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/library-go/logger"
	"github.com/urfave/cli"
	"os"
	"strings"
	"time"
)

func CmdSync() cli.Command {
	return cli.Command{
		Name:      "sync",
		Usage:     "同步备份功能(Beta)",
		UsageText: cmder.App().Name + " sync",
		Description: `
	备份功能。指定本地目录和对应的一个网盘目录，以备份文件。
	备份功能支持一下三种模式：
	1. upload 
       备份本地文件，即上传本地文件到网盘，始终保持本地文件有一个完整的备份在网盘

	2. download 
       备份云盘文件，即下载网盘文件到本地，始终保持网盘的文件有一个完整的备份在本地

	3. sync 
       双向备份，保持网盘文件和本地文件严格一致

`,
		Category: "阿里云盘",
		Before:   cmder.ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if config.Config.ActiveUser() == nil {
				fmt.Println("未登录账号")
				return nil
			}
			blockSize := int64(c.Int("bs") * 1024)
			RunSync(blockSize)
			return nil
		},
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "log",
				Usage: "开启log输出",
			},
			cli.IntFlag{
				Name:  "bs",
				Usage: "block size，上传分片大小，单位KB。推荐值：1024 ~ 10240",
				Value: 10240,
			},
		},
	}
}

func RunSync(uploadBlockSize int64) {
	activeUser := GetActiveUser()
	panClient := activeUser.PanClient()

	// pan token expired checker
	go func() {
		for {
			time.Sleep(time.Duration(1) * time.Minute)
			if RefreshTokenInNeed(activeUser) {
				logger.Verboseln("update access token for sync task")
				panClient.UpdateToken(activeUser.WebToken)
			}
		}
	}()

	syncFolderRootPath := config.GetSyncDriveDir()
	if b, e := utils.PathExists(syncFolderRootPath); e == nil {
		if !b {
			os.MkdirAll(syncFolderRootPath, 0600)
		}
	}

	fmt.Println("启动同步备份进程")
	syncMgr := syncdrive.NewSyncTaskManager(activeUser.DriveList.GetFileDriveId(), panClient, syncFolderRootPath)
	if _, e := syncMgr.Start(); e != nil {
		fmt.Println("启动任务失败：", e)
		return
	}
	c := ""
	for strings.ToLower(c) != "y" {
		fmt.Print("本命令不会退出，如需要结束同步备份进程请输入y，然后按Enter键进行停止：")
		fmt.Scan(&c)
	}
	fmt.Println("正在停止同步备份任务，请稍等...")
	syncMgr.Stop()
}