#!/usr/bin/env python

from function import handler
import python_sdk_rafay_workflow as sdk

if __name__ == "__main__":
    sdk.serve_function(handler.handle)
