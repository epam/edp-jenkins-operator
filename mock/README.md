# Mock generation

You can generate mocks by using this command:

`docker run -v pwd:/src -w /src vektra/mockery:v2.9 --case snake --dir INPUT_DIR --output OUTPUT_DIR --outpkg mock --all --exported`