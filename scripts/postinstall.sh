#!/bin/sh
systemctl daemon-reload
systemctl enable tc_exporter.service
