# Skill: Deploy App

## Objective

Your goal as DevOps is to intelligently package and deploy the application, supporting both remote NAS environments and local hosting setups based on the project configuration.

## Instructions

1. **Deployment Type Detection**:
   * Inspect the workspace to see if it is destined for remote NAS deployment:
     * Check for `skills-codelab/app_build/scripts/deploy-nas.sh` or `scripts/deploy-nas.sh` or `scripts/deploy-qnap.sh`.
     * Check for any deployment script in the project root (e.g. `deploy.sh`).
     * Inspect `Technical_Specification.md` and `Deployment_Manual.md` for references to QNAP or NAS.

2. **Execute Deployment**:
   * **If a QNAP/NAS Deploy Script is detected**:
     * Make sure the script is executable.
     * Check if a secrets configuration file is required (e.g. `notifyhub-secrets.json`) and prompt the user if it's missing from `data/` or target directories.
     * Execute the deploy script with the appropriate sub-command (e.g. `./skills-codelab/app_build/scripts/deploy-nas.sh deploy` or `./scripts/deploy-qnap.sh deploy`).
     * Monitor output to verify successful container loading and service startup.
   * **If no remote deployment script is detected (Fallback to Local)**:
     * Inspect files in `app_build/` to figure out the stack.
     * Navigate into `app_build/` and run the appropriate dependencies installation command (e.g., `npm install`, `pip install -r requirements.txt`).
     * Start the server locally (e.g., `npm run dev`, `python3 app.py`).

3. **Report Status**:
   * Output the clickable URL of the deployed application:
     * For NAS deployments: the remote API/health endpoints on the QNAP NAS.
     * For Local deployments: the clickable `localhost` link.
   * Celebrate a successful launch!
