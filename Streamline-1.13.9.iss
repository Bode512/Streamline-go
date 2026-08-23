; Streamline Desktop - Inno Setup release 1.13.9
; Este script no ejecuta la aplicacion durante la instalacion.
; Requiere Inno Setup 6.x y un desktop.exe previamente compilado.

#define AppName "Streamline"
#define AppVersion "1.13.9"
#define AppPublisher "Streamline"
#define AppExeName "desktop.exe"
#define BuildDir "desktop\build\bin"
#define IconAsset "desktop\build\appicon.png"
#define OutputDir "release"

#ifexist "desktop\build\bin\desktop.exe"
#else
  #error "Falta desktop\build\bin\desktop.exe. Ejecuta wails build antes de compilar el instalador."
#endif

[Setup]
AppId={{B9A4DE8A-5B88-4C0B-9D4E-113900000001}}
AppName={#AppName}
AppVersion={#AppVersion}
AppVerName={#AppName} {#AppVersion}
AppPublisher={#AppPublisher}
AppPublisherURL=https://github.com/
DefaultDirName={localappdata}\Streamline
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
ArchitecturesInstallIn64BitMode=x64compatible
ArchitecturesAllowed=x86compatible x64compatible
OutputDir={#OutputDir}
OutputBaseFilename=Streamline-Setup-{#AppVersion}
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
UninstallDisplayName={#AppName} {#AppVersion}
UninstallDisplayIcon={app}\{#AppExeName}
; Inno Setup no acepta PNG como icono de SetupIconFile.
; El PNG se instala como recurso y el ejecutable Wails conserva su icono.
; Wails usa WebView2 del sistema. Windows 10/11 normalmente ya lo incluye.
; Si se distribuye en equipos sin WebView2, debe añadirse su instalador oficial.

[Languages]
Name: "spanish"; MessagesFile: "compiler:Languages\Spanish.isl"
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "Crear un acceso directo en el escritorio"; GroupDescription: "Accesos directos:"; Flags: unchecked
Name: "startmenuicon"; Description: "Crear un acceso directo en el menu Inicio"; GroupDescription: "Accesos directos:"; Flags: checkedonce

[Files]
; El unico ejecutable visible contiene Core Go, HandBrakeCLI y la interfaz Wails.
Source: "{#BuildDir}\{#AppExeName}"; DestDir: "{app}"; Flags: ignoreversion
; Se incluyen DLL adicionales si el build las necesitara en el futuro.
Source: "{#BuildDir}\*.dll"; DestDir: "{app}"; Flags: ignoreversion skipifsourcedoesntexist
; El logo PNG queda disponible junto a la aplicacion.
Source: "{#IconAsset}"; DestDir: "{app}\assets"; Flags: ignoreversion skipifsourcedoesntexist

[Dirs]
Name: "{app}\videos"
Name: "{app}\convertidos"

[Icons]
Name: "{autodesktop}\{#AppName}"; Filename: "{app}\{#AppExeName}"; Tasks: desktopicon
Name: "{group}\{#AppName}"; Filename: "{app}\{#AppExeName}"; Tasks: startmenuicon
Name: "{group}\Desinstalar {#AppName}"; Filename: "{uninstallexe}"; Tasks: startmenuicon

[Run]
Filename: "{app}\{#AppExeName}"; Description: "Abrir {#AppName}"; Flags: nowait postinstall skipifsilent unchecked

[UninstallDelete]
; No se eliminan videos, conversiones ni la base de datos del usuario.
Type: filesandordirs; Name: "{app}\assets"

