# 获取日期格式化的版本号
#$ver = $( Get-Date -Format "yy.MMdd" )
$ver = "24.0116"
$commit = "${ver}.25ce0e7"
# 使用 sed 替换 config/init.go 中的版本号
sed -i "/const version = */c const version = ""$commit""" config/init.go

# 配置 GOPROXY 环境变量
$env:GOPROXY = "https://goproxy.io,direct"


function BuildAndPackageWin
{
    $ENV:CGO_ENABLED = 0
    $ENV:GOOS = "windows"
    $ENV:GOARCH = "amd64"

    $target = "target/bookget-$ver.$ENV:GOOS-$ENV:GOARCH"
    mkdir -Force $target

    cp -Recurse "target/bookget-gui/*" $target
    #     rm -Force "$target/bookget-gui.pdb"

    go build -o "$target/bookget.exe" .

    cp "config.ini" "$target/config.ini"
    cp "target\dezoomify-rs\x86_64-windows\dezoomify-rs.exe" "$target/dezoomify-rs.exe"

    cd "target/"
    7z a -t7z "bookget-$ver.$ENV:GOOS-$ENV:GOARCH.7z" "bookget-$ver.$ENV:GOOS-$ENV:GOARCH"
    cd ../

    rm -Force -Recurse $target
}

function BuildAndPackageLinux
{
    $ENV:CGO_ENABLED = 0
    $ENV:GOOS = "linux"
    $ENV:GOARCH = "amd64"

    $target = "target/bookget-$ver.$ENV:GOOS-$ENV:GOARCH"
    mkdir -Force $target

    go build -o "$target/bookget" .

    cp "config.ini" "$target/config.ini"
    cp "target\dezoomify-rs\x86_64-linux\dezoomify-rs" "$target/dezoomify-rs"

    cd "target/"
    7z a -t7z "bookget-$ver.$ENV:GOOS-$ENV:GOARCH.7z" "bookget-$ver.$ENV:GOOS-$ENV:GOARCH"
    cd ../

    rm -Force -Recurse $target
}

function BuildAndPackageDarwin
{
    $ENV:CGO_ENABLED = 0
    $ENV:GOOS = "darwin"
    $ENV:GOARCH = "amd64"

    $target = "target/bookget-$ver.$ENV:GOOS-$ENV:GOARCH"
    mkdir -Force $target

    go build -o "$target/bookget" .

    cp "config.ini" "$target/config.ini"
    cp "target\dezoomify-rs\x86_64-apple\dezoomify-rs" "$target/dezoomify-rs"

    cd "target/"
    7z a -t7z "bookget-$ver.$ENV:GOOS-$ENV:GOARCH.7z" "bookget-$ver.$ENV:GOOS-$ENV:GOARCH"
    cd ../

    rm -Force -Recurse $target
}

function BuildAndPackageDarwinArm
{
    $ENV:CGO_ENABLED = 0
    $ENV:GOOS = "darwin"
    $ENV:GOARCH = "arm64"

    $target = "target/bookget-$ver.$ENV:GOOS-$ENV:GOARCH"
    mkdir -Force $target

    go build -o "$target/bookget" .

    cp "config.ini" "$target/config.ini"
    cp "target\dezoomify-rs\aarch64-apple\dezoomify-rs" "$target/dezoomify-rs"

    cd "target/"
    7z a -t7z "bookget-$ver.$ENV:GOOS-$ENV:GOARCH.7z" "bookget-$ver.$ENV:GOOS-$ENV:GOARCH"
    cd ../

    rm -Force -Recurse $target
}

# 调用函数
BuildAndPackageWin
# BuildAndPackageLinux
# BuildAndPackageDarwin
# BuildAndPackageDarwinArm

