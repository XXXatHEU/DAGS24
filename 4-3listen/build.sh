#!/bin/bash
rm blockchain
#rm *.db
#rm ./otherpeer/* -rf
rm err* -f
rm wallet.dat blockchain.db -rf
cp ./beifen/* ./
go build -o blockchain *.go
cp ./blockchain ./otherpeer/

./blockchain
