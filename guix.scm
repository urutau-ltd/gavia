(use-modules (gnu packages golang)
             (gnu packages golang-build)
             (gnu packages golang-xyz)
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

(define-public gavia-dev
  (package
   (name "gavia-dev")
   (version "dev")
   (source
    (local-file %project-directory
                "gavia-checkout"
                #:recursive? #t
                #:select? (git-predicate %project-directory)))
   (build-system go-build-system)
   (arguments
    (list
     #:go go-1.26
     #:install-source? #f
     ;; Guix's go-build-system forces GO111MODULE=off.
     ;; This project depends on Aile route mounting that behaves incorrectly
     ;; under GOPATH mode and turns integration tests into 404s during the
     ;; Guix check phase, even though the package compiles successfully.
     #:tests? #f
     #:import-path "codeberg.org/urutau-ltd/gavia"
     #:unpack-path "codeberg.org/urutau-ltd/gavia"))
   (native-inputs
    (list go-codeberg-org-urutau-ltd-aile-v2
          go-github-com-google-uuid
          go-modernc-org-sqlite))
   (home-page "https://codeberg.org/urutau-ltd/gavia/")
   (synopsis "Infrastructure inventory web tool")
   (description "Gavia is a self-hosted Go application for keeping 
infrastructure inventory, billing notes, settings, and small operational 
dashboards in one place. It uses server-rendered HTML with Aile, HTMX, 
Hyperscript, Missing.css, and SQLite.")
   (license license:agpl3)))

gavia-dev
