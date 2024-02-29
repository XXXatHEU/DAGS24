../build.sh
cp ../blockchain ./
./blockchain
rm  blockchain.db -rf
cp ../beifen/blockchain.db  ./

rm err* -rf
