
# devmails

DevServer to test & render MJML templates with Go templates.


## Install

```shell
curl -sSL https://git.io/fjuM7 | sudo bash
```


## Usage

Run the app in the folder where you have a `src` folder with the templates and a `data` folder with the custom data. See the [testdata](testdata) folder for an example of the structure.

```shell
devmails
```


Change the location of any of the folders if needed:

```shell
devmails -src my/folder/src -output other/folder/output -data other-other/folder/data
```


To run the generator a single time and close afterwards without watching changes in the files:

```shell
devmails -watch=false
```


## Contributing

You can make pull requests or create issues in GitHub. Any code you send should be formatted using `make gofmt`.


## Running tests

Run the tests:

```shell
make test
```


## License

[MIT License](LICENSE)
