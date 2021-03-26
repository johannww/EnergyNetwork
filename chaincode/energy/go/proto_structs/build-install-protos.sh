#this script compiles the 'struct.proto' in the folder of this script
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
protoc --proto_path=$DIR --go_out=$DIR --go_opt=paths=source_relative  structs.proto