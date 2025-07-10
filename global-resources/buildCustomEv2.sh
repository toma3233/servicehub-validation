outputDir=$1
echo "Test Global Resources" # TODO (Christine): Add test script for global resources' names
echo "Package config file"
cd config
zip -r config.zip ./*
mv config.zip $outputDir/Ev2Specs/Build
cd ..