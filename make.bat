@echo off
setlocal

if "%1"=="" goto :usage

if /I "%1"=="fmt" goto :fmt
if /I "%1"=="test" goto :test
if /I "%1"=="vet" goto :vet
if /I "%1"=="build" goto :build
if /I "%1"=="check" goto :check
if /I "%1"=="check-all" goto :check-all

echo Unknown target: %1
goto :usage

:fmt
go fmt ./...
exit /b %errorlevel%

:test
go test ./...
exit /b %errorlevel%

:vet
go vet ./...
exit /b %errorlevel%

:build
go build ./...
exit /b %errorlevel%

:check
call "%~f0" fmt
if errorlevel 1 exit /b %errorlevel%
call "%~f0" test
if errorlevel 1 exit /b %errorlevel%
call "%~f0" vet
if errorlevel 1 exit /b %errorlevel%
call "%~f0" build
exit /b %errorlevel%

:check-all
call "%~f0" fmt
if errorlevel 1 exit /b %errorlevel%
call "%~f0" test
if errorlevel 1 exit /b %errorlevel%
call "%~f0" vet
if errorlevel 1 exit /b %errorlevel%
call "%~f0" build
exit /b %errorlevel%

:usage
echo Usage: make ^<fmt^|test^|vet^|build^|check^|check-all^>
exit /b 2
