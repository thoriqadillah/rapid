[app]
title = rapid
project_dir = .
input_file = main.py
exec_directory = ./dist
project_file = 
icon = /home/thoriqadillah/workspace/rapid/.venv/lib/python3.13/site-packages/PySide6/scripts/deploy_lib/pyside_icon.jpg

[python]
python_path = /home/thoriqadillah/workspace/rapid/.venv/bin/python3
packages = Nuitka==4.0
android_packages = buildozer==1.5.0,cython==0.29.33

[qt]
qml_files = qml/Main.qml
excluded_qml_plugins = QtCharts,QtQuick3D,QtSensors,QtTest,QtWebEngine
modules = Core,DBus,Gui,Network,OpenGL,Qml,QmlMeta,QmlModels,QmlWorkerScript,Quick,QuickControls2,QuickTemplates2,Widgets
plugins = accessiblebridge,egldeviceintegrations,generic,iconengines,imageformats,networkaccess,networkinformation,platforminputcontexts,platforms,platforms/darwin,platformthemes,qmllint,qmltooling,scenegraph,tls,vectorimageformats,wayland-decoration-client,wayland-graphics-integration-client,wayland-shell-integration,xcbglintegrations

[android]
wheel_pyside = 
wheel_shiboken = 
plugins = 

[nuitka]
macos.permissions = 
mode = standalone
extra_args = --verbose --noinclude-qt-translations --static-libpython=no

[buildozer]
mode = debug
recipe_dir = 
jars_dir = 
ndk_path = 
sdk_path = 
local_libs = 
arch = 

