ver="1.2.1"

mkdir -p target/bookget-${ver}.linux/
mkdir -p target/bookget-${ver}.macOS/
mkdir -p target/bookget-${ver}.macOS-arm64/

#CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o target/bookget-${ver}.windows/bookget.exe .
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o target/bookget-${ver}.linux/bookget .
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o target/bookget-${ver}.macOS/bookget .
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o target/bookget-${ver}.macOS-arm64/bookget .


cp cookie.txt target/bookget-${ver}.linux/cookie.txt
cp cookie.txt target/bookget-${ver}.macOS/cookie.txt
cp cookie.txt target/bookget-${ver}.macOS-arm64/cookie.txt
#cp cookie.txt target/bookget-${ver}.windows/cookie.txt

cp target/dezoomify-rs/x86_64-linux/dezoomify-rs target/bookget-${ver}.linux/dezoomify-rs
cp target/dezoomify-rs/x86_64-apple/dezoomify-rs target/bookget-${ver}.macOS/dezoomify-rs
cp target/dezoomify-rs/aarch64-apple/dezoomify-rs target/bookget-${ver}.macOS-arm64/dezoomify-rs


cd target/
#7za a -t7z bookget-${ver}.windows.7z bookget-${ver}.windows
tar cjf bookget-${ver}.linux.tar.bz2 bookget-${ver}.linux
tar cjf bookget-${ver}.macOS.tar.bz2 bookget-${ver}.macOS
tar cjf bookget-${ver}.macOS-arm64.tar.bz2 bookget-${ver}.macOS-arm64


