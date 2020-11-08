cd varnam-editor
call yarn install

SET VUE_PUBLIC_PATH=/
call yarn build

cd ..
rd /s /q ui
move varnam-editor\dist ui
