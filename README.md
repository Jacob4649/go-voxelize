# go-voxelize

Small Go script to voxelize LiDAR data stored in .LAS format.

## Usage

```shell
./voxelize input.las output.csv <number of chunks for reading & processing> <number of concurrent processing> <point density to be considered a voxel>
```

## Licensing

Originally used [lidario](https://github.com/jblindsay/lidario), but due to lack of support for concurrent reading, a small modification to the library had to be made. Now transitioning away from the library completely.
