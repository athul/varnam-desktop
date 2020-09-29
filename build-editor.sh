cd varnam-editor
yarn install

export VUE_PUBLIC_PATH="/"
yarn build

rm -rf $(realpath ../)/ui
mv dist $(realpath ../)/ui
