# Varnam Desktop App

Cross-platform desktop app for Varnam. Uses [varnam-editor](https://github.com/thetronjohnson/varnam-editor) as GUI frontend.

## Development

* Clone the repo.
* Install dependencies :
  Debian or Ubuntu based systems :
  ```
  sudo apt install libwebkit2gtk-4.0-37
  ```
  Windows : Include [these DLLs](https://github.com/webview/webview/tree/master/dll) in repo folder
  Mac : Safari Browser should be installed
* Build `libvarnam` (On Windows, Visual Studio is needed to make the DLLs) :
```
git submodule update --init --recursive
cd libvarnam/libvarnam && cmake . && make varnamstatic && ln -s $(realpath libvarnamstatic.a) libvarnam.a && cd -
```
* Build `varnam` :

```
make deps
make editor
make build
```

* Run :

```
./varnam
```

Thanks to [stuffbin](https://github.com/knadh/stuffbin), HTML+CSS+JS files are all merged into one big binary (`varnam`).

## Usage

`varnam` bundles with `varnamd`, the HTTP server to interface with varnam library. By default, the server runs on `127.0.0.1:8123`. See [varnamd](https://github.com/varnamproject/varnamd) for more details.