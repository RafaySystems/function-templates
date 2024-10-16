import unittest
import logging
from . import handler
import os
import time
import random
from python_sdk_rafay_workflow import sdk

AWS_ACCESS_KEY_ID=os.environ.get('AWS_ACCESS_KEY_ID')
AWS_SECRET_ACCESS_KEY=os.environ.get('AWS_SECRET_ACCESS_KEY')
AWS_REGION=os.environ.get('AWS_REGION')
CF_TEMPLATE=os.fspath('examples/python/cloudformation/cfs3.template')

class TestCloudFormation(unittest.TestCase):
    def __init__(self, *args, **kwargs):
        super(TestCloudFormation, self).__init__(*args, **kwargs)
        
        
    def setUp(self) -> None:
        # set up
        return

    def tearDown(self) -> None:
        # tear down
        return

    def test_cloudformation(self):
        # test cloudformation
        with open(CF_TEMPLATE, 'r', encoding='utf-8') as file:
            template = file.read()

        # generate random 4 char string
        rand = ''.join(random.choices('abcdefghijklmnopqrstuvwxyz', k=4))

        stack_name = f'avinash-test-s3-{rand}'

        request = {
            'stack_name': stack_name,
            'aws_access_key': AWS_ACCESS_KEY_ID,
            'aws_secret_key': AWS_SECRET_ACCESS_KEY,
            'aws_region': AWS_REGION,
            'action': 'deploy',
            'template': template,
            'template_inputs': [
                    {
                        'ParameterKey': 'BucketName',
                        'ParameterValue': stack_name,
                    }
                ]
        }

        # create
        response = {}
        for i in range(10):
            try:
                response = handler.handle(logging.Logger('test'), request)
            except sdk.ExecuteAgainException as e:
                # retry
                time.sleep(5)
                request['previous'] = e.data
                continue
            break

        self.assertEqual(response['status'], 'success')
        
        # destroy
        request['action'] = 'destroy'
        response = handler.handle(logging.Logger('test'), request)



if __name__ == "__main__":
    unittest.main()