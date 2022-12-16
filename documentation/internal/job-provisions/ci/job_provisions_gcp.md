# Job Provisions for GCP Issues

Since GCP Load balancers have strict limitation of the header size in 16Kb, EDP Jenkins script should be as small as possible.
In EDP, Job provisions are the largest scripts and they cannot be reduced in a programming way, but there is a solution.
Simply compress the Job provisions with the Groovy zip library before run and then decompress them in Jenkins in runtime.

>**INFO**: The decompressed provisions are stored in the 'build/configs/job-provisions/ci/' folder.

## How to Create New Archived Provisions

To create new archived provisions, follow the steps below:

1. Modify the provision code in the 'build/configs/job-provisions/ci/' folder.
2. Create a job in any Jenkins with the following code:
    ```groovy
    import java.util.zip.GZIPInputStream
    import java.util.zip.GZIPOutputStream

    def zip(String s){
	    def targetStream = new ByteArrayOutputStream()
	    def zipStream = new GZIPOutputStream(targetStream)
	    zipStream.write(s.getBytes('UTF-8'))
	    zipStream.close()
	    def zippedBytes = targetStream.toByteArray()
	    targetStream.close()
	    return zippedBytes.encodeBase64()
    }

    String inString = 'whatever'
    String zipString = zip(inString)
    println zipString
    ```
3. Put the new provision code as the 'inString' value in the created job.
4. Run the job to get the compressed provision code.
5. Copy the compressed provision code to the corresponding provision in the 'build/configs/job-provisions/ci' file as the 'compressedScriptText' value.

## Groovy Code for Compression / Decompression

Familiarize yourself with the Groovy code **sample** for compression and decompression:

```groovy
import java.util.zip.GZIPInputStream
import java.util.zip.GZIPOutputStream

def zip(String s){
	def targetStream = new ByteArrayOutputStream()
	def zipStream = new GZIPOutputStream(targetStream)
	zipStream.write(s.getBytes('UTF-8'))
	zipStream.close()
	def zippedBytes = targetStream.toByteArray()
	targetStream.close()
	return zippedBytes.encodeBase64()
}

def unzip(String compressed){
	def inflaterStream = new GZIPInputStream(new ByteArrayInputStream(compressed.decodeBase64()))
    def uncompressedStr = inflaterStream.getText('UTF-8')
    return uncompressedStr
}

String inString = 'whatever'
String zipString = zip(inString)
String unzippedString = unzip(zipString)
assert(inString == unzippedString)

```
