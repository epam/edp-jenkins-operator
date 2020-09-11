## Job provisions for GCP issues

Since GCP Load balancers have hard limitation of 16Kb header size, EDP Jenkins scrips should be as small as possible. Job provisions are our biggest scripts and they could not be made smaller in programming way. The solution is to compress them with Groove zip library before run ande decompress in Jenkins in runtime.

Decompressed provisions are stored in 'documentation/internal/job-provisions/ci' folder.  

### How to create new archived provisions

1. Modify provision code at 'documentation/internal/job-provisions/ci/'

2. Create job in any Jenkins with the following code:
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

3. Put new provision code as 'inString' value in created job

4. Run job to get compressed provision code

5. Copy compressed provision code in corresponding provision at 'build/configs/job-provisions/ci' as 'compressedScriptText' value

### Sample Groovy code for compression / decompression

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