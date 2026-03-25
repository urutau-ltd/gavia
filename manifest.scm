;; Lo que sigue es un "manifest" equivalente a la línea de comando que
;; introdujo. Puede almacenarlo dentro de un archivo que pudiese pasar a
;; cualquier comando 'guix' que acepte una opción '--manifest' (o -m).

(specifications->manifest
 (list "go@1.26"
       "gopls"
       "make"
       "gcc-toolchain"
       "podman"
       "podman-compose"
       "curl"
       "sqlite"
       "go-golang-org-x-tools-godoc"
       "go-github-com-go-jose-go-jose-v4"
       "go-github-com-google-uuid"
       "git"))
