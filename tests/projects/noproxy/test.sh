#!/bin/bash

# This is intended to be called from wercker/test-all.sh, which sets the required environment variables
# if you run this file directly, you need to set $wercker, $workingDir and $testDir
# as a convenience, if these are not set then assume we're running from the local directory 
if [ -z ${wercker} ]; then wercker=$PWD/../../../wercker; fi
if [ -z ${workingDir} ]; then workingDir=$PWD/../../../.werckertests; mkdir -p "$workingDir"; fi
if [ -z ${testsDir} ]; then testsDir=$PWD/..; fi

# alpine, http_proxy is set, no_proxy is set, HTTP_PROXY is not set, NO_PROXY is not set
testNoProxy1 () {
  testName=noproxy1
  testDir=$testsDir/noproxy
  printf "testing %s... " "$testName"
  #
  export X_testDir=$testDir # we will mount this inside the pipeline container
  export X_real_http_proxy=$http_proxy
  export X_real_no_proxy=$no_proxy
  export http_proxy=http://someproxy:80
  export no_proxy=foo.com
  export X_expected_http_proxy_1=$http_proxy
  export X_expected_http_proxy_2=$http_proxy
  export X_expected_http_proxy_3=$http_proxy
  export X_expected_http_proxy_4=$http_proxy
  export X_expected_no_proxy_1=nginx2,nginx1,$no_proxy
  export X_expected_no_proxy_2=nginx3,$X_expected_no_proxy_1
  export X_expected_no_proxy_3=nginx4,$X_expected_no_proxy_2
  export X_expected_no_proxy_4=$X_expected_no_proxy_3  
  export X_REAL_HTTP_PROXY=$HTTP_PROXY
  export X_REAL_NO_PROXY=$NO_PROXY
  export HTTP_PROXY=
  export NO_PROXY=   
  export X_expected_HTTP_PROXY_1=$HTTP_PROXY
  export X_expected_HTTP_PROXY_2=$HTTP_PROXY
  export X_expected_HTTP_PROXY_3=$HTTP_PROXY
  export X_expected_HTTP_PROXY_4=$HTTP_PROXY
  export X_expected_NO_PROXY_1=
  export X_expected_NO_PROXY_2=
  export X_expected_NO_PROXY_3=
  export X_expected_NO_PROXY_4=
  rm -f $testDir/failed 
  $wercker build "$testDir" --pipeline testalpine --enable-volumes --working-dir "$workingDir" &> "${workingDir}/${testName}.log"
  if [ $? -ne 0 ] || [ -f $testDir/failed ]; then
    rm -f $testDir/failed 
    printf "failed\n"
    if [ "${workingDir}/${testName}.log" ]; then
      cat "${workingDir}/${testName}.log"
    fi
    return 1
  fi 
  printf "passed\n"
  export http_proxy=$X_real_http_proxy
  export no_proxy=$X_real_no_proxy
  return 0
}

# alpine, http_proxy is not set, no_proxy is not set, HTTP_PROXY is set, NO_PROXY is set
testNoProxy2 () {
  testName=noproxy2
  testDir=$testsDir/noproxy
  printf "testing %s... " "$testName"
  #
  export X_testDir=$testDir # we will mount this inside the pipeline container
  export X_real_http_proxy=$http_proxy
  export X_real_no_proxy=$no_proxy
  export http_proxy=
  export no_proxy=
  export X_expected_http_proxy_1=$http_proxy
  export X_expected_http_proxy_2=$http_proxy
  export X_expected_http_proxy_3=$http_proxy
  export X_expected_http_proxy_4=$http_proxy
  export X_expected_no_proxy_1=
  export X_expected_no_proxy_2=
  export X_expected_no_proxy_3=
  export X_expected_no_proxy_4=  
  export X_REAL_HTTP_PROXY=$HTTP_PROXY
  export X_REAL_NO_PROXY=$NO_PROXY
  export HTTP_PROXY=http://someproxy:80
  export NO_PROXY=foo.com  
  export X_expected_HTTP_PROXY_1=$HTTP_PROXY
  export X_expected_HTTP_PROXY_2=$HTTP_PROXY
  export X_expected_HTTP_PROXY_3=$HTTP_PROXY
  export X_expected_HTTP_PROXY_4=$HTTP_PROXY
  export X_expected_NO_PROXY_1=nginx2,nginx1,$NO_PROXY
  export X_expected_NO_PROXY_2=nginx3,$X_expected_NO_PROXY_1
  export X_expected_NO_PROXY_3=nginx4,$X_expected_NO_PROXY_2
  export X_expected_NO_PROXY_4=$X_expected_NO_PROXY_3
  rm -f $testDir/failed 
  $wercker build "$testDir" --pipeline testalpine --enable-volumes --working-dir "$workingDir" &> "${workingDir}/${testName}.log"
  if [ $? -ne 0 ] || [ -f $testDir/failed ]; then
    rm -f $testDir/failed 
    printf "failed\n"
    if [ "${workingDir}/${testName}.log" ]; then
      cat "${workingDir}/${testName}.log"
    fi
    return 1
  fi 
  printf "passed\n"
  export http_proxy=$X_real_http_proxy
  export no_proxy=$X_real_no_proxy  
  return 0
}

# ubuntu, http_proxy is set, no_proxy is set, HTTP_PROXY is not set, NO_PROXY is not set
testNoProxy3 () {
  testName=noproxy3
  testDir=$testsDir/noproxy
  printf "testing %s... " "$testName"
  #
  export X_testDir=$testDir # we will mount this inside the pipeline container
  export X_real_http_proxy=$http_proxy
  export X_real_no_proxy=$no_proxy
  export http_proxy=http://someproxy:80
  export no_proxy=foo.com
  export X_expected_http_proxy_1=$http_proxy
  export X_expected_http_proxy_2=$http_proxy
  export X_expected_http_proxy_3=$http_proxy
  export X_expected_http_proxy_4=$http_proxy
  export X_expected_no_proxy_1=nginx2,nginx1,$no_proxy
  export X_expected_no_proxy_2=nginx3,$X_expected_no_proxy_1
  export X_expected_no_proxy_3=nginx4,$X_expected_no_proxy_2
  export X_expected_no_proxy_4=$X_expected_no_proxy_3  
  export X_REAL_HTTP_PROXY=$HTTP_PROXY
  export X_REAL_NO_PROXY=$NO_PROXY
  export HTTP_PROXY=
  export NO_PROXY=   
  export X_expected_HTTP_PROXY_1=$HTTP_PROXY
  export X_expected_HTTP_PROXY_2=$HTTP_PROXY
  export X_expected_HTTP_PROXY_3=$HTTP_PROXY
  export X_expected_HTTP_PROXY_4=$HTTP_PROXY
  export X_expected_NO_PROXY_1=
  export X_expected_NO_PROXY_2=
  export X_expected_NO_PROXY_3=
  export X_expected_NO_PROXY_4=
  rm -f $testDir/failed 
  $wercker build "$testDir" --pipeline testubuntu --enable-volumes --working-dir "$workingDir" &> "${workingDir}/${testName}.log"
  if [ $? -ne 0 ] || [ -f $testDir/failed ]; then
    rm -f $testDir/failed 
    printf "failed\n"
    if [ "${workingDir}/${testName}.log" ]; then
      cat "${workingDir}/${testName}.log"
    fi
    return 1
  fi 
  printf "passed\n"
  export http_proxy=$X_real_http_proxy
  export no_proxy=$X_real_no_proxy  
  return 0
}


testNoProxyAll () {
  testNoProxy1 || return 1 
  testNoProxy2 || return 1 
  testNoProxy3 || return 1 
}

testNoProxyAll
