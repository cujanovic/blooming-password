# Blooming Password

A program that implements the [NIST 800-63-3b Banned Password Check](https://nvlpubs.nist.gov/nistpubs/SpecialPublications/NIST.SP.800-63b.pdf) using a [bloom filter](https://dl.acm.org/citation.cfm?doid=362686.362692) built from the [Have I been pwned](https://haveibeenpwned.com/Passwords) SHA1 password hash list. The Have I Been Pwned v6 list contains 572611621 password hashes and is 24GB uncompressed (as of 13 Jun 2020). A bloom filter of this list is only 982MB(with 1 in 1000 False Positive Rate https://hur.st/bloomfilter/?n=572611621&p=0.001&m=&k=10) and will fit entirely into memory on a virtual machine or Docker container with just 4GB of RAM.

## Why a Bloom Filter?

It's one of the simplest, smallest and fastest data structures for this task. Bloom filters have constant time O(1) performance (where K is the constant) for insertion and lookup. K is the number of times a password is hashed. Bloom filters can easily handle billions of banned password hashes with very modest resources. When a test for membership returns 418 (I'm a teapot) then it's safe to use that password.

## Partial SHA1 Hashes

SHA1 hashes are 20 bytes of raw binary data and thus typically hex encoded for a total of 40 characters. Blooming Password uses just the first 16 hex encoded characters of the hashes to build the bloom filter and to test the filter for membership. The program rejects complete hashes if they are sent. False positive rates in the bloom filter are not impacted by the shortening of the SHA1 password hashes. The cardinality of the set is unchanged. The FP rate is .001 (1 in 1000 https://hur.st/bloomfilter/?n=572611621&p=0.001&m=&k=10). You may verify the cardinality is unchanged after truncating the hashes.

```
  $ wc -l pwned-passwords-sha1-ordered-by-count-v6.txt
  572611621 pwned-passwords-sha1-ordered-by-count-v6.txt

  $ sort -T /tmp/ -u 1-16-pwned-passwords-sha1-ordered-by-count-v6.txt | wc -l
  572611621
```

## How to Construct the Partial SHA1 Hash List

```
  $ 7z e pwned-passwords-sha1-ordered-by-count-v6.7z

  $ cut -c 1-16 pwned-passwords-sha1-ordered-by-count-v6.txt > 1-16-pwned-passwords-sha1-ordered-by-count-v6.txt

  $ head 1-16-pwned-passwords-sha1-ordered-by-count-v6.txt
	7C4A8D09CA3762AF
	F7C3BC1D808E0473
	B1B3773A05C0ED01
	5BAA61E4C9B93F3F
	3D4F2BF07DC1BE38
  ...
```

## How to Create the Bloom Filter

```
  $ tools/blooming-password-filter-create /path/to/1-16-pwned-passwords-sha1-ordered-by-count-v6.txt /path/to/1-16-pwned-passwords-sha1-ordered-by-count-v6.filter
```

## Test the Bloom Filter for Membership

Send the first 16 characters of the hex encoded SHA1 hash to the Blooming Password program. Some examples using curl:

  * curl -4 https://server-name:server-port/check/sha1/0123456789ABCDEF
  * curl -6 https://server-name:server-port/check/sha1/F7C3BC1D808E0473
  * curl -4 https://server-name:server-port/check/sha1/$(echo -n "secret123" | shasum | cut -c 1-16)

## Return Codes

  * [200](https://server-name:server-port/check/sha1/F7C3BC1D808E0473) - OK. The hash is probably in the bloom filter.
  * [400](https://server-name:server-port/check/sha1/PASSWORD) - Bad request. The client sent a bad request.
  * [418](https://server-name:server-port/check/sha1/0123456789ABCDEF) - I'm a teapot. The hash is definitely not in the bloom filter.

  Note: If the value is in the filter, the server will return a 200 status code, otherwise a 418 (I'm a teapot). The latter is used to be distinguishable from a 404 that you might receive for other reasons (e.g. misconfigured servers).

## Benchmark

Server used is AWS t3.medium instance and one of the previous version(5) of HaveIBeenPwned Pwned Passwords list, which was latest when test was performed.

```
root@ip-10-20-19-7:~# ./vegeta attack -targets=benchmark-test.txt -rate=50 -duration=60s | ./vegeta report -type=text
Requests      [total, rate, throughput]         3000, 50.02, 5.35
Duration      [total, attack, wait]             59.98s, 59.98s, 503.579µs
Latencies     [min, mean, 50, 90, 95, 99, max]  359.298µs, 602.367µs, 548.977µs, 633.498µs, 661.026µs, 798.467µs, 58.958ms
Bytes In      [total, mean]                     72491, 24.16
Bytes Out     [total, mean]                     0, 0.00
Success       [ratio]                           10.70%
Status Codes  [code:count]                      200:321  418:2679
Error Set:
418 I'm a teapot
root@ip-10-20-19-7:~# ./vegeta attack -targets=benchmark-test.txt -rate=100 -duration=60s | ./vegeta report -type=text
Requests      [total, rate, throughput]         6000, 100.02, 10.70
Duration      [total, attack, wait]             59.99s, 59.99s, 370.664µs
Latencies     [min, mean, 50, 90, 95, 99, max]  314.501µs, 515.238µs, 451.698µs, 546.85µs, 593.354µs, 939.655µs, 59.576ms
Bytes In      [total, mean]                     144982, 24.16
Bytes Out     [total, mean]                     0, 0.00
Success       [ratio]                           10.70%
Status Codes  [code:count]                      200:642  418:5358
Error Set:
418 I'm a teapot
root@ip-10-20-19-7:~# ./vegeta attack -targets=benchmark-test.txt -rate=200 -duration=60s | ./vegeta report -type=text
Requests      [total, rate, throughput]         12000, 200.02, 21.40
Duration      [total, attack, wait]             59.995s, 59.995s, 329.075µs
Latencies     [min, mean, 50, 90, 95, 99, max]  289.114µs, 671.934µs, 359.274µs, 444.906µs, 506.313µs, 3.164ms, 102.726ms
Bytes In      [total, mean]                     289964, 24.16
Bytes Out     [total, mean]                     0, 0.00
Success       [ratio]                           10.70%
Status Codes  [code:count]                      200:1284  418:10716
Error Set:
418 I'm a teapot
root@ip-10-20-19-7:~# ./vegeta attack -targets=benchmark-test.txt -rate=400 -duration=60s | ./vegeta report -type=text
Requests      [total, rate, throughput]         24000, 400.02, 42.85
Duration      [total, attack, wait]             59.998s, 59.998s, 879.735µs
Latencies     [min, mean, 50, 90, 95, 99, max]  271.21µs, 481.118ms, 360.287µs, 2.016s, 2.995s, 3.475s, 3.867s
Bytes In      [total, mean]                     580241, 24.18
Bytes Out     [total, mean]                     0, 0.00
Success       [ratio]                           10.71%
Status Codes  [code:count]                      200:2571  418:21429
Error Set:
418 I'm a teapot
root@ip-10-20-19-7:~# ./vegeta attack -targets=benchmark-test.txt -rate=800 -duration=60s | ./vegeta report -type=text
Requests      [total, rate, throughput]         48000, 800.02, 85.70
Duration      [total, attack, wait]             59.999s, 59.999s, 362.645µs
Latencies     [min, mean, 50, 90, 95, 99, max]  265.386µs, 819.493ms, 335.5µs, 3.358s, 4.978s, 6.553s, 7.988s
Bytes In      [total, mean]                     1160482, 24.18
Bytes Out     [total, mean]                     0, 0.00
Success       [ratio]                           10.71%
Status Codes  [code:count]                      200:5142  418:42858
Error Set:
418 I'm a teapot
```

# Blooming Password - Create filter

The Create filter program creates a new bloom filter. It takes two arguments.

  1. Path to the text file containing partial SHA1 hashes (one hash per line). The partial SHA1 hashes must be **UPPERCASE**.
  2. Path to where you'd like to save the bloom filter.

## What the partial SHA1 hash file should look like

```bash
head 1-16-pwned-passwords-sha1-ordered-by-count-v6.txt
7C4A8D09CA3762AF
F7C3BC1D808E0473
B1B3773A05C0ED01
5BAA61E4C9B93F3F
3D4F2BF07DC1BE38
...
```

## How to run Create filter

```bash
./tools/blooming-password-filter-create /path/to/1-16-pwned-passwords-sha1-ordered-by-count-v6.txt /path/to/1-16-pwned-passwords-sha1-ordered-by-count-v6.filter
```

## Docker
Add - https://docs.docker.com/engine/reference/run/#runtime-constraints-on-resources

```
docker build -t blooming-password .
docker run --cap-drop=all --security-opt=no-new-privileges:true --read-only -v /etc/ssl/:/etc/ssl/:ro --publish 9379:9379 --detach --name BP-Server blooming-password
docker logs BP-Server
```



## Notes
  * The Blooming Password **blooming-password-server.go** program reads the bloom filter produced by **tools/blooming-password-filter-create.go**.
  * Blooming Password is written in [Go](https://golang.org).
  * It uses [willf's excellent bloom filter](https://github.com/willf/bloom) implementation.
