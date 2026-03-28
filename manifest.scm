;; Lo que sigue es un "manifest" equivalente a la línea de comando que
;; introdujo. Puede almacenarlo dentro de un archivo que pudiese pasar a
;; cualquier comando 'guix' que acepte una opción '--manifest' (o -m).
(use-modules (gnu packages golang)
             (guix packages)
             (guix git-download)
             (guix build-system go)
             (guix utils)
             ((guix licenses)
              #:prefix license:)
             (guix gexp))

(define %project-directory
  (dirname (current-filename)))

(define-public go-codeberg-org-urutau-ltd-aile-v2
  (package
    (name "go-codeberg-org-urutau-ltd-aile-v2")
    (version "2.1.1")
    (source
     (origin
      (method git-fetch)
      (uri (git-reference
            (url "https://codeberg.org/urutau-ltd/aile.git")
            (commit (string-append "v" version))))
      (file-name (git-file-name name version))
      (sha256
       (base32 "1bxm7j53z2pjivgg4x0hynvji1cw1fbcllv5c1pcg0qa4dl1083z"))))
    (build-system go-build-system)
    (arguments
     (list
      #:go go-1.26
      ;; Guix forces GO111MODULE=off on the go-build-system
      ;; This package uses Go 1.22+ net/http ServeMux method/path patterns.
      ;; Under Guix's GOPATH mode, tests fail with 404s, and downstream
      ;; consumers may observe the same behaviour if blindly importing
      ;; and building from guix import go.
      #:tests? #f
      #:import-path "codeberg.org/urutau-ltd/aile/v2"))
    (home-page "https://codeberg.org/urutau-ltd/aile")
    (synopsis "Small http runtime for Go")
    (description
     "Package aile provides a small HTTP runtime for Go built around the
standard library. This package definition builds the local repository checkout
with the Go 1.26 toolchain.")
    (license license:agpl3)))

(concatenate-manifests (list (specifications->manifest (list "go@1.26"
                                                             "gopls"
                                                             "make"
                                                             "gcc-toolchain"
                                                             "podman"
                                                             "podman-compose"
                                                             "curl"
                                                             "sqlite"
                                                             "go-golang-org-x-tools-godoc"
                                                             "go-github-com-google-uuid"
                                                             "go-modernc-org-sqlite"
                                                             "git"))
                             (packages->manifest (list go-codeberg-org-urutau-ltd-aile-v2))))


