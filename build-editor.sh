cd varnam-editor
npm install

export VUE_PUBLIC_PATH="/"
npm run build

rm -rf $(realpath ../)/ui
mv dist $(realpath ../)/ui
