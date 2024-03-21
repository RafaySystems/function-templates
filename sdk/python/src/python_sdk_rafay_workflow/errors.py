
ERROR_CODE_EXECUTE_AGAIN = 1
ERROR_CODE_FAILED = 2
ERROR_CODE_TRANSIENT = 3

class FunctionException(Exception):
    def __init__(self, message, error_code=ERROR_CODE_FAILED):
        self.message = message
        self.error_code = error_code

class ExecuteAgainException(FunctionException):
    def __init__(self, message, **kwargs):
        self.data = kwargs
        super().__init__("function: execute again: {}".format(message), ERROR_CODE_EXECUTE_AGAIN)

class FailedException(FunctionException):
    def __init__(self, message):
        super().__init__("function: failed: {}".format(message), ERROR_CODE_FAILED)

class TransientException(FunctionException):
    def __init__(self, message):
        super().__init__("function: transient error: {}".format(message), ERROR_CODE_TRANSIENT)