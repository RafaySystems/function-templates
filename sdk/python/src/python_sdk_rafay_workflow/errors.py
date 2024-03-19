
ERROR_CODE_EXECUTE_AGAIN = 1
ERROR_CODE_FAILED = 2
ERROR_CODE_TRANSIENT = 3

class FunctionException(Exception):
    def __init__(self, message, code=ERROR_CODE_FAILED):
        self.message = message
        self.code = code

class ExecuteAgainException(FunctionException):
    def __init__(self, message):
        super().__init__("function: execute again: {}".format(message), ERROR_CODE_EXECUTE_AGAIN)

class FailedException(FunctionException):
    def __init__(self, message):
        super().__init__("function: failed: {}".format(message), ERROR_CODE_FAILED)

class TransientException(FunctionException):
    def __init__(self, message):
        super().__init__("function: transient error: {}".format(message), ERROR_CODE_TRANSIENT)