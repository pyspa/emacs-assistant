;;; pyspa.el --- This package contains a variety of features -*- coding: utf-8; lexical-binding: t; -*-

;; Copyright (C) 2020 Yutaka Matsubara

;; Author: Yutaka Matsubara (yutaka.matsubara@gmail.com)
;; Homepage: https://github.com/pyspa/emacs-is-not-dead
;; Package-Version: 0.0.1
;; Package-Requires: ((s "1.12.0") (dash "2.17.0") (dash-functional "1.2.0") (emacs "24"))

(require 'cl-lib)
(require 'pcase)
(require 'thingatpt)
(require 's)
(require 'dash)
(require 'dash-functional)

(module-load (expand-file-name "libpyspaemacs.so" user-emacs-directory))

(defun pyspa-echo (arg)
  (interactive "smsg: ")
  (pyspa/echo arg))
