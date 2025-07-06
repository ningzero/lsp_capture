# lsp-capture
Capture the communication between lsp client and server and record them in logs respectively.

# usage
## build
```
go build
```
## run
modify the command your language client uses to start your language server to the following
```
lsp-capture language-server.bin some_args
```