:gate message
tool\protoc --go_out="..\message" --go_opt=paths=source_relative --go-grpc_out="..\message" --go-grpc_opt=paths=source_relative *.proto
pause
