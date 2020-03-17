#!/bin/bash
# Orignal https://gist.github.com/jmervine/7d3f455e923cf2ac3c9e
# usage: ./golang-crosscompile-build.bash

#Clear Sceeen
printf "\033c"

# Get Version out of target then replace . with _
versiond=$(go run *.go -version)
version=${versiond//./_}
#Remove White Space
version=${version// /}
versiond=${versiond// /}
#platforms="darwin/386 darwin/amd64 freebsd/386 freebsd/amd64 freebsd/arm linux/386 linux/amd64 linux/arm windows/386 windows/amd64"
platforms="windows/386 windows/amd64 linux/386 linux/amd64 linux/arm darwin/386 darwin/amd64"
printf " ---- Building Azure User Import $versiond ---- \n"

rm -rf "release/"
mkdir release

printf "Replace Version Variable\n"

printf "\n"
for platform in ${platforms}
do
    split=(${platform//\// })
    goos=${split[0]}
    os=${split[0]}
    goarch=${split[1]}
    arch=${split[1]}
    output=goAzure2HUserImport
    package=goAzure2HUserImport
    # add exe to windows output
    [[ "windows" == "$goos" ]] && output="$output.exe"
    [[ "windows" == "$goos" ]] && os="win"
    [[ "386" == "$goarch" ]] && arch="x86"
    [[ "amd64" == "$goarch" ]] && arch="x64"

    printf "Platform: $goos - $goarch \n"

    destination="builds/$goos/$goarch/$output"

    printf "Go Build\n"
    GOOS=$goos GOARCH=$goarch go build  -o $destination $target

    printf "Copy Source Files\n"
    #Copy Source to Build Dir
    cp LICENSE.md "builds/$goos/$goarch/LICENSE.md"
    cp README.md "builds/$goos/$goarch/README.md"
    cp conf.json "builds/$goos/$goarch/conf.json"

    printf "Build Zip \n"
    cd "builds/$goos/$goarch/"
    if [ $os == "darwin" ]; then
        os="osx"
    fi
    zip -r "${package}_${os}_${arch}_v${version}.zip" $output LICENSE.md README.md conf.json > /dev/null
    cp "${package}_${os}_${arch}_v${version}.zip" "../../../release/${package}_${os}_${arch}_v${version}.zip"
    cd "../../../"
    printf "\n"
done
printf "Clean Up \n"
rm -rf "builds/"
printf "Build Complete \n"
printf "\n"