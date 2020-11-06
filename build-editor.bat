cd varnam-editor
yarn install

SET VUE_PUBLIC_PATH=/
yarn build

SET workingDir=%~dp0
rd /s /q %workingDir:~0,-1%\..\..\ui
move dist %workingDir:~0,-1%\..\..\ui
