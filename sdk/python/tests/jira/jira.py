from typing import *
import requests
from requests.auth import HTTPBasicAuth
import json
import time
import sys
import os
from logging import Logger
from python_sdk_rafay_workflow import sdk

JIRA_USER_EMAIL=os.environ.get('JIRA_USER_EMAIL')
JIRA_APPROVER_EMAIL=os.environ.get('JIRA_APPROVER_EMAIL','')
JIRA_ADMIN_USERNAME=os.environ.get('JIRA_ADMIN_USERNAME')
JIRA_API_TOKEN=os.environ.get('JIRA_API_TOKEN')
JIRA_HOST_NAME=os.environ.get('JIRA_HOST_NAME')
ENVIRONMENT_NAME=os.environ.get('ENVIRONMENT_NAME')
ENVIRONMENT_TEMPLATE_NAME=os.environ.get('ENVIRONMENT_TEMPLATE_NAME')
JIRA_PROJECT=os.environ.get('JIRA_PROJECT')

def create_issue(accountId,projectId,approverId):
    url = "https://"+JIRA_HOST_NAME+"/rest/api/3/issue"

    auth = HTTPBasicAuth(JIRA_ADMIN_USERNAME, JIRA_API_TOKEN)

    headers = {
        "Accept": "application/json",
        "Content-Type": "application/json"
    }

    payload = json.dumps( {
        "fields": {
            "assignee": {
                "accountId": approverId
            },
            "description": {
                "content": [
                    {
                        "content": [
                            {
                                "text": ENVIRONMENT_NAME+" for "+JIRA_USER_EMAIL,
                                "type": "text"
                            }
                        ],
                        "type": "paragraph"
                    }
                ],
                "type": "doc",
                "version": 1
            },
            "issuetype": {
                "id": "10002"
            },
            "project": {
                "id": projectId
            },
            # "reporter": {
            #     "accountId": accountId
            # },
            "summary": ENVIRONMENT_TEMPLATE_NAME+" for "+JIRA_USER_EMAIL,
        },
        "update": {}
    } )

    response = requests.request(
        "POST",
        url,
        data=payload,
        headers=headers,
        auth=auth
    )

    #print(json.dumps(json.loads(response.text), sort_keys=True, indent=4, separators=(",", ": ")))
    return json.loads(response.text)['id']

def get_user(email):
    url = "https://"+JIRA_HOST_NAME+"/rest/api/3/user/search"
    auth = HTTPBasicAuth(JIRA_ADMIN_USERNAME, JIRA_API_TOKEN)
    headers = {
        "Accept": "application/json"
    }

    query = {
        'query': email
    }
    response = requests.request(
        "GET",
        url,
        headers=headers,
        params=query,
        auth=auth
    )

    #print(json.dumps(json.loads(response.text), sort_keys=True, indent=4, separators=(",", ": ")))
    return(json.loads(response.text)[0]['accountId'])

def get_status(jira_id):
    url = "https://"+JIRA_HOST_NAME+"/rest/api/3/issue/"+jira_id
    status = "Wait"
    auth = HTTPBasicAuth(JIRA_ADMIN_USERNAME, JIRA_API_TOKEN)
    headers = {
        "Accept": "application/json"
    }
    response = requests.request(
        "GET",
        url,
        headers=headers,
        auth=auth
    )

    #print(json.dumps(json.loads(response.text), sort_keys=True, indent=4, separators=(",", ": ")))
    status = json.loads(response.text)['fields']['status']['name']
    print(status)
    return status

def get_project_id(project):
    url = "https://" + JIRA_HOST_NAME + "/rest/api/3/project/" + project
    auth = HTTPBasicAuth(JIRA_ADMIN_USERNAME, JIRA_API_TOKEN)
    headers = {
        "Accept": "application/json"
    }
    response = requests.request(
        "GET",
        url,
        headers=headers,
        auth=auth
    )
    #print(json.dumps(json.loads(response.text), sort_keys=True, indent=4, separators=(",", ": ")))
    return(json.loads(response.text)['id'])

def check_if_ticket_exists(ticket_id=""):
    query=ENVIRONMENT_NAME+" for "+JIRA_USER_EMAIL
    url = "https://" + JIRA_HOST_NAME + "/rest/api/3/issue/" + ticket_id
    auth = HTTPBasicAuth(JIRA_ADMIN_USERNAME, JIRA_API_TOKEN)
    headers = {
        "Accept": "application/json"
    }
    response = requests.request(
        "GET",
        url,
        headers=headers,
        auth=auth
    )
    #print(json.dumps(json.loads(response.text), sort_keys=True, indent=4, separators=(",", ": ")))
    if response.status_code >=200 and response.status_code < 300:
        return True
    else:
        return False

def approve_issue(jira_id):
    url = "https://"+JIRA_HOST_NAME+"/rest/api/3/issue/"+jira_id+"/transitions"
    auth = HTTPBasicAuth(JIRA_ADMIN_USERNAME, JIRA_API_TOKEN)
    headers = {
        "Accept": "application/json",
        "Content-Type": "application/json"
    }

    payload = json.dumps( {
        "transition": {
            "id": 971
        }
    } )

    response = requests.request(
        "POST",
        url,
        data=payload,
        headers=headers,
        auth=auth
    )

    response.raise_for_status()

def handle(logger: Logger,request: Dict[str, Any]) -> Dict[str, Any]:
    try:
        counter = request['previous'].get('counter', 0) if 'previous' in request else 0
        logger.info("Checking if ticket exists")
        id = request['previous'].get('ticket_id', '') if 'previous' in request else ''
        if id:
            # exists=check_if_ticket_exists(ticket_id=request["previous"]["ticket_id"])
            status=get_status(id)
        else:
            logger.info("Creating ticket")
            accountId=get_user(JIRA_USER_EMAIL)
            if JIRA_APPROVER_EMAIL == '':
                approverId="-1"
            else:
                approverId=get_user(JIRA_APPROVER_EMAIL)
            projectId=get_project_id(JIRA_PROJECT)
            id=create_issue(accountId,projectId,approverId)
            logger.info(f"Ticket created {id}")
            status=get_status(id)
    except ConnectionError as e:
        logger.error(f"ConnectionError: {str(e)}")
        raise sdk.TransientException(f"ConnectionError: {str(e)}")
    except Exception as e:
        logger.error(f"FailedException: {str(e)}")
        raise sdk.FailedException(f"FailedException: {str(e)}")

    if status == 'Approved':
        logger.info("Ticket approved")
        return {"status": "Approved"}
    if status == 'Declined':
        logger.info("Ticket declined")
        return {"status": "Declined"}
    else:
        logger.info("Waiting for ticket to be approved or declined")
        raise sdk.ExecuteAgainException("Please wait for the ticket to be approved or declined", ticket_id=id, counter=counter+1)

