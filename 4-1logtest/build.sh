#!/bin/bash
rm test -f
#rm *.db

go build -o test *.go
./test
