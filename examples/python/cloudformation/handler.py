from typing import Dict, Any
from logging import Logger
from python_sdk_rafay_workflow import sdk
import boto3

def handle(logger: Logger,request: Dict[str, Any]) -> Dict[str, Any]:
    logger.info("received request")

    handler = CloudFormationHandler(logger, request)
    resp = handler.handle()

    return resp

class CloudFormationHandler:
    def __init__(self, logger: Logger, request: Dict[str, Any]):
        self.logger = logger
        self.request = request

        # Get the stack name from the request
        self.stack_name = request.get('stack_name')
        if not self.stack_name:
            raise sdk.FailedException("stack_name is required")

        self.access_key = request.get('aws_access_key')
        if not self.access_key:
            raise sdk.FailedException("aws_access_key is required")

        self.secret_key = request.get('aws_secret_key')
        if not self.secret_key:
            raise sdk.FailedException("aws_secret_key is required")

        self.region = request.get('aws_region')
        if not self.region:
            raise sdk.FailedException("aws_region is required")

        self.action = request.get('action')
        if not self.action:
            action = 'deploy'

        self.template = request.get('template')
        if not self.template:
            raise sdk.FailedException("template is required")

        self.template_inputs = request.get('template_inputs')
        if not self.template_inputs:
            raise sdk.FailedException("template_inputs is required")

        # create aws session
        self.session = boto3.Session(
            aws_access_key_id=self.access_key,
            aws_secret_access_key=self.secret_key,
            region_name=self.region
        )

        # create cloudformation client
        self.client = self.session.client('cloudformation')

    def handle(self) -> Dict[str, Any]:
        if self.action == 'deploy':
            return self.deploy()
        elif self.action == 'destroy':
            return self.destroy()
        else:
            raise sdk.FailedException("Invalid action")

    def deploy(self) -> Dict[str, Any]:
        self.logger.info("deploying stack")

        # check if previous response exists
        previous = self.request.get('previous')
        if previous:
            stack_id = previous.get('StackId', None)
            self.logger.info(f"previous response exists for stack {stack_id}")
            stack = self.deploy_status(stack_id)
            if stack:
                return {
                    'status': 'success',
                    'outputs': stack.get('Outputs', [])
                }

        # check if stack exists
        stack = self.existing_stack(self.stack_name)

        if stack:
            return self.update()
        else:
            return self.create()

    def create(self) -> Dict[str, Any]:
        self.logger.info("create stack")

        # deploy the stack
        try:
            response = self.client.create_stack(
                StackName=self.stack_name,
                TemplateBody=self.template,
                Parameters=self.template_inputs,
            )
        except self.client.exceptions.ClientError as e:
            raise sdk.FailedException(f"Error occurred while creating stack {e}")

        raise sdk.ExecuteAgainException("Stack creation in progress", StackId=response.get('StackId'))

    def update(self) -> Dict[str, Any]:
        self.logger.info("updating stack")

        # update the stack
        try:
            response = self.client.update_stack(
                StackName=self.stack_name,
                TemplateBody=self.template,
                Parameters=self.template_inputs,
            )
        except self.client.exceptions.ClientError as e:
            if e.response['Error']['Code'] == 'ValidationError' and 'No updates are to be performed' in e.response['Error']['Message']:
                return
            else:
                raise sdk.FailedException(f"Error occurred while updating stack {e}")

        raise sdk.ExecuteAgainException("Stack update in progress", StackId=response.get('StackId'))

    def destroy(self) -> Dict[str, Any]:
        self.logger.info("destroying stack")
        
        stack = self.existing_stack(self.stack_name)
        if not stack:
            return
        # destroy the stack
        response = self.client.delete_stack(
            StackName=self.stack_name
        )
        return response

    def deploy_status(self, stack_id) -> Dict[str, Any]:
        self.logger.info("getting stack status")

        # get stack status
        resp = self.client.describe_stacks(StackName=stack_id)
        if resp:
            stack = resp['Stacks'][0]
            if stack['StackStatus'] in ['CREATE_IN_PROGRESS', 'UPDATE_IN_PROGRESS', 'DELETE_IN_PROGRESS']:
                raise sdk.ExecuteAgainException("Stack operation in progress", StackId=stack_id)
            elif stack['StackStatus'] in ['CREATE_COMPLETE', 'UPDATE_COMPLETE', 'DELETE_COMPLETE']:
                return stack
            else:
                raise sdk.FailedException(f"Stack operation failed {stack['StackStatus']}")
                
        return stack

    def existing_stack(self, stack_name) -> Dict[str, Any]:
        self.logger.info("checking stack exists")

        # check if stack exists
        try:
            stack = self.client.describe_stacks(StackName=stack_name)
        except self.client.exceptions.ClientError as e:
            if e.response['Error']['Code'] == 'ValidationError':
                stack = None
            else:
                raise sdk.TransientException(f"Error occurred while checking stack {e}")

        return stack