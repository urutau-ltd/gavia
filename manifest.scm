;; Lo que sigue es un "manifest" equivalente a la línea de comando que
;; introdujo. Puede almacenarlo dentro de un archivo que pudiese pasar a
;; cualquier comando 'guix' que acepte una opción '--manifest' (o -m).

(specifications->manifest
 (list "go@1.25"
       "gopls"
       "make"
       "gcc-toolchain"
       "podman"
       "podman-compose"
       "curl"
       "go-golang-org-x-tools-godoc"
       "git"))
