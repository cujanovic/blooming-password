package main

import (
	"bufio"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"github.com/willf/bloom"
)

// general config
var (
	FalsePositiveRate     float64
	NumberOfElements      float64
	NumberOfHashFunctions uint
	Bloomm                float64
	BloomfilterFile       *bloom.BloomFilter
)

// ParseConfig : parse the configuration file
func ParseConfig() {
	log.Printf("======================================================================================================")
	log.Printf("Using default config: " + "./configs/blooming-password-filter-create.conf")
	log.Printf("======================================================================================================")
	viper.AddConfigPath("./configs")
	viper.SetConfigName("blooming-password-filter-create.conf")
	viper.SetConfigType("json")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
	FalsePositiveRate = viper.GetFloat64("FalsePositiveRate")
	NumberOfElements = viper.GetFloat64("NumberOfElements")
	NumberOfHashFunctions = viper.GetUint("NumberOfHashFunctions")
}

// load creates a bloom filter from the partial hashes
// and saves the filter to a file. The hashes must be UPPERCASE or the checks will fail.
func main() {
	usage := "blooming-password-filter-create /path/to/1-16-pwned-passwords-sha1-ordered-by-count-v6.txt /path/to/1-16-pwned-passwords-sha1-ordered-by-count-v6.filter"
	if len(os.Args) != 3 {
		fmt.Println(usage)
		return
	}
	// parse config file
	ParseConfig()
	hashFile := os.Args[1]
	filterFile := os.Args[2]
	// create Bloom filter
	Bloomm = math.Ceil((NumberOfElements * math.Log(FalsePositiveRate)) / math.Log(1.0/math.Pow(2.0, math.Log(2.0))))
	BloomfilterFile = bloom.New(uint(Bloomm), NumberOfHashFunctions)
	// populate the bloom filter
	file, err := os.Open(filepath.Clean(hashFile))
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		BloomfilterFile.Add(scanner.Bytes())
	}
	err = scanner.Err()
	if err != nil {
		log.Fatal(err)
	}
	// save the bloom filter to a file
	f, err := os.Create(filterFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	bytesWritten, err := BloomfilterFile.WriteTo(f)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("bytes written to Bloom filter("+filterFile+"): %d\n", bytesWritten)
}
