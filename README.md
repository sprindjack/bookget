# bookget

bookget 数字图书馆下载工具，目前支持约50+个数字图书馆。

### 使用说明：

1. 打开 https://github.com/deweizhu/bookget/releases 下载最新版。
1. [必读]使用手册wiki https://github.com/deweizhu/bookget/wiki
1. 此项目代码仅供学习研究使用，欢迎有能力的朋友git clone 代码二次开发维护您自己的版本。
1. 请诸位养成保持下载最新代码的习惯。

### 从源码构建编译
从源码构建，仅对计算机程序员参考。普通用户可直接跳过阅读。   
阅读 golang 官方文档 https://golang.google.cn/doc/install ，给您的电脑安装 golang 开发环境。
```shell
git clone --depth=1 https://github.com/deweizhu/bookget.git
cd bookget
go build .
```
- 源码构建的 MacOS/Linux 版，与releases发布版一致。
- 源码构建的 Windows 版缺少webview支持，部分网站无法下载。例如：韩国国家图书馆、香港中文大学图书馆。

### 特别推荐：

[书格](https://new.shuge.org/) 有品格的数字古籍图书馆   
[![书格](https://new.shuge.org/wp-content/themes/artview/images/layout/logo.png "书格")](https://new.shuge.org/)


