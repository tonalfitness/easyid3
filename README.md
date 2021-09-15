# easyid3
This library parses ID3v2 blocks from a reader. It doesn't enforce specific
tags listed in some of the specs so reads pretty much anything that matches the [structure](https://id3.org/id3v2.4.0-structure) including partial data. It does minimal error checking for validity so it may parse some invalid structures if the ID3 is malformed (this is on purpose).
 

