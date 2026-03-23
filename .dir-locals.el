;;; .dir-locals.el --- Per directory variables for Emacs -*- lexical-binding: t; -*-
;;
;; SPDX-License-Identifier: AGPL-3.0-or-later
;; Copyright © 2024-2025 Urutau-Ltd <softwarelibre@urutau-ltd.org>
;;
;;   , _ ,      _    _            _                     _ _      _
;;  ( o o )    | |  | |          | |                   | | |    | |
;; /'` ' `'\   | |  | |_ __ _   _| |_ __ _ _   _ ______| | |_ __| |
;; |'''''''|   | |  | | '__| | | | __/ _` | | | |______| | __/ _` |
;; |\\'''//|   | |__| | |  | |_| | || (_| | |_| |      | | || (_| |
;;    """       \____/|_|   \__,_|\__\__,_|\__,_|      |_|\__\__,_|
;;
;; This program is free software: you can redistribute it and/or modify
;; it under the terms of the GNU Affero General Public License as published by
;; the Free Software Foundation, either version 3 of the License, or (at
;; your option) any later version.
;;
;; This program is distributed in the hope that it will be useful, but
;; WITHOUT ANY WARRANTY; without even the implied warranty of
;; MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
;; General Public License for more details.
;;
;; You should have received a copy of the GNU Affero General Public License
;; along with this program. If not, see <https://www.gnu.org/licenses/>.
;;
;;; Commentary:
;;
;; This file contains per-directory variables for easier Emacs development
;; settings. Mainly for the stack used inside this repository which should be
;; one of a static site.
;;
;; This file shouldn't change much anyways.
;;
;;; Code:
((nil
  . ((fill-column . 80)
     (tab-width   .  8)
     ;; For use with 'bug-reference-prog-mode'
     (bug-reference-url-format . "https://github.com/urutau-ltd/gavia/issues/%s")
     (bug-reference-bug-regexp
      . "\\(#\\([0-9]+\\)\\)")
     (geiser-insert-actual-lambda . nil)
     (eval . (progn
               (add-to-list 'completion-ignored-extensions ".go")
               (add-to-list 'auto-mode-alist '("\\.mdx\\'" . markdown-mode))))))
 (emacs-lisp-mode . ((indent-tabs-mode . nil)))
 (markdown-mode    . ((indent-tabs-mode . nil)
                     (fill-column      . 80)))
 (gfm-mode . ((indent-tabs-mode . nil)
              (fill-column . 80)))
 (typescript-mode .
                  ((indent-tabs-mode . nil)
                   (tab-width . 4)
                   (fill-column . 80)))
 (web-mode .
           ((indent-tabs-mode . nil)
            (tab-width . 4)
            (fill-column . 80)))
 (json-mode .
            ((indent-tabs-mode . nil)
             (tab-width . 4)
             (fill-column . 80)))
 (scheme-mode .
              ((geiser-guile-binary . ("guile"))
               (eval . (put 'sxml-match 'scheme-indent-function 1))))
 (makefile-gmake-mode .
           ((indent-tabs-mode . t)
            (tab-width . 4)
            (fill-column . 80))))
