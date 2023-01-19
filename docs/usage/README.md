This guide is about Gardener Logging, how it is organized and how to use the dashboard to view the log data of Kubernetes clusters.

# Cluster level logging
Log data is fundamental for the successful operation activities of Kubernetes landscapes. It is used for investigating problems and monitoring cluster activity.

Cluster level logging is the recommended way to collect and store log data for Kubernetes cluster components. With cluster level logging the log data is externalized
in a logging backend where the log lifecycle management is independent from the lifecycle management of the Kubernetes resources.

Cluster level logging is not available by default with [Kubernetes](https://kubernetes.io/docs/concepts/cluster-administration/logging/#cluster-level-logging-architectures) and consumers have to additionally implement it.
The Kubernetes project only provides basic logging capabilities via `kubectl logs` where the kubelet keeps one terminated container with its logs.
When a pod is evicted from the node, all corresponding containers are also evicted, along with their logs.
This is why the default log storage solution is considered short-lived and not sufficient when you want to properly operate a Kubernetes environment.

Gardener, as an advanced Kubernetes management solution, follows the general recommendations and offers a cluster level logging solution to ensure proper log storage for all managed Kubernetes resources.
The log management is setup when a new cluster is created.
Log collection is organized using [fluent-bit](https://fluentbit.io).
Log storage and search is organized using Vali.
Log visualization is available using Plutono that is deployed with predefined dashboard and visualization for every shoot cluster.


Using Kubernetes operators can benefit from different capabilities like accessing the logs for
already terminated containers and performing fast and sophisticated search queries for investigating long-lasting or recurring problems based on logs from a long period of time.

In this guide, you will find out how to explore the log data for your clusters.

## Exploring logs

The sections below describe how access Plutono and use it to view the log data of your Kubernetes cluster.

### Accessing Plutono
1. On the Gardener dashboard, choose **CLUSTERS** > [YOUR-CLUSTER] > **OVERVIEW** > **Logging and Monitoring**.
![Navigate to Logging and Monitoring Tile](images/gardener-dashboard-logging.png)

2. Use the link in the **Logging and Monitoring** tile to open the Plutono dashboard.
3. Enter the login credentials shown in the **Logging and Monitoring** tile to log in the Plutono dashboard.
The default values of the credentials for Plutono are:
- Username : `admin`
- Password : `admin`
![Login Screen](images/login-credentials.png)

Upon successful login you will be asked to changing the default password.
**Note:** These credentials are shared among all operators. Changing the default password will affect their access. You can safely skip this step.
![Button to Skip Password Change](images/skip-password-change.png)

### Using Plutono

There are two ways to explore log messages in Plutono.

#### Predefined Dashboards
The first one is to use the predefined dashboards.
1. Go to the **Home** tab.
2. Choose which dashboard to open.
The dashboards that contain log visualizations for the different Plutono deployments are:

  * Garden Plutono
    * Pod Logs
    * Extensions
    * Systemd Logs
  * User Plutono
    * Kubernetes Control Plane Status
  * Operator Plutono
    * Kubernetes Pods
    * Kubernetes Control Plane Status

    ![Dashboard Navigator](images/dashboards.png)

#### Explore tab
The second one is to use the **Explore** tab.

To enable this option you need to authenticate in front of the Plutono UI.
1. Choose the login button (bottom left corner).
![Login Button on Plutono Home Screen](images/login-button.png)

2. Log in following the steps described in the [Acccessing Plutono](#accessing-plutono) section.
3. Choose the ***Explore*** tab (upper left side of the screen).
![Plutono Explore Tab](images/explore-logs.png)
You can create a custom log filters based on the predefined labels used in `Vali`.
The following properties can be managed in the `Explore` tab:
- `Datasource` (top left corner) should be set on Vali
- `Timerange` (top right corner) is used to filter logs over a different period of time
- `Label Selector` (top left corner) is used to filter logs based on the `Vali`'s labels and their values.
For example:
`pod_name="kube-apiserver-1234-1234"` or you can use a regular expression (regex): `pod_name=~"kube-apiserver.+"`
- `Severity` (left side of the screen). This option is used to filter log messages with specific severity.

4. Click on **Run Query** (top right corner) and the log messages, which fulfil the list of selected properties above, will be displayed.
