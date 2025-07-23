# OneCert Certificate Workflow

[OneCert](https://aka.ms/onecert) is a robust Registration Authority (RA) service offered in Azure that simplifies the process of enrolling TLS certificates. It serves as a trusted intermediary between Azure Key Vault (AKV) and Datacenter Secrets Management Service (DSMS), connecting them with integrated Certificate Authorities (CAs). OneCert ensures the validity of certificate enrollment requests and strictly adheres to a first-come, first-served policy for domain registrations. With OneCert, services can secure exclusivity for their domain names, as no other entity can obtain certificates for the same domain without explicit permission. OneCert seamlessly integrates with various CAs across sovereign clouds, providing a secure and efficient solution for managing certificates in Azure environments.

This document explains how certificates issued by OneCert are managed and consumed in a Kubernetes environment, with a focus on Azure Key Vault integration, the Secrets Store CSI driver, and RBAC requirements. It also covers where these certificates are used (e.g., Geneva, UserRP Ingress) and provides links to relevant documentation.

---

## 1. Certificate Lifecycle Overview

### 1.1. [OneCert Registration](https://eng.ms/docs/products/onecert-certificates-key-vault-and-dsms/key-vault-dsms/onecert/docs/registering-a-domain-in-onecert)
- Certificates are registered through the [OneCert portal](https://aka.ms/onecert), this can only be done manually through SAW.

Userrp registration example: 
![Userrp OneCert Register](images/domain_registration_screenshot.png)

### 1.2. [Certificate Creation in Azure Key Vault](https://eng.ms/docs/products/onecert-certificates-key-vault-and-dsms/key-vault-dsms/onecert/docs/requesting-a-onecert-certificate-with-keyvault?tabs=azure-powershell)
- The OneCert Registration Authority (RA) is represented in Key Vault as two providers: OneCertV2-PrivateCA and OneCertV2-PublicCA. Select the OneCertV2-PrivateCA provider to create certificates from the Private Issuer CA selected in the OneCert domain registration. Select the OneCertV2-PublicCA to create certificates from the the Public Issuer CA selected in the OneCert domain registration.
- **How the certificate is stored in Key Vault:**
    - In our workflow, a Bicep module provisions a PowerShell deployment script that runs during resources deployment. This script uses the integrated OneCert Registration Authority (RA) to request a certificate for the registered domain and stores the resulting certificate directly in Azure Key Vault.
    - The script specifies the correct Key Vault instance, certificate name, and selects the appropriate OneCertV2 provider (PrivateCA or PublicCA) based on the domain registration.
    - The certificate is created and managed in Key Vault, with renewal policies and access controls defined as part of the deployment. This ensures the certificate is securely stored and available for syncing to Kubernetes via the CSI driver.

### 1.3. Syncing Certificate from Key Vault to Kubernetes with CSI Driver
- The [Secrets Store CSI driver](https://learn.microsoft.com/azure/aks/csi-secrets-store-driver) is configured to sync secrets/certificates from Key Vault into Kubernetes as native secrets.
- A `SecretProviderClass` custom resource defines which Key Vault objects to sync.
- The CSI driver runs as a privileged pod in the cluster and requires appropriate RBAC permissions (see below).


### 1.4. Consuming the Certificate
- The synced Kubernetes secret can be mounted to a pod or directly referenced by an ingress controller (e.g., for TLS termination in UserRP Ingress).
- **Geneva**: The monitoring agent (`mdsd`) on the AKS cluster uses a key vault certificate as a secret in the mounted volume to authenticate with Geneva.
- **UserRP Ingress**: The ingress controller references the secret for HTTPS endpoints.

---

## 2. RBAC: Why ClusterRole and ClusterRoleBinding Are Needed

- The Secrets Store CSI driver must access and manage secrets across namespaces, which requires cluster-wide permissions.
- The provided `ClusterRole` grants the driver permissions to get, list, watch, create, update, patch, and delete Kubernetes secrets.
- The `ClusterRoleBinding` binds this role to the service account which is used in the CSI driver.
    - In Kubernetes, every pod runs under a service account, which determines its permissions. The Secrets Store CSI driver runs as a set of pods (DaemonSet) in the cluster. These pods are configured to use a specific service account. The `ClusterRoleBinding` links the required permissions (from the `ClusterRole`) to this service account, allowing the CSI driver pods to perform actions (like creating or updating secrets) across namespaces. If you are using the AKS-managed addon, this service account and its binding are created and managed for you automatically.
- Without these, the CSI driver cannot sync secrets from Key Vault or create/update secrets in target namespaces.

**Reference:**
- [Kubernetes RBAC documentation](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#rolebinding-example)

### 2.1. Note on AKS Secrets Provider Addon RBAC Management

- When using the AKS Secrets Store CSI driver via the managed AKS secrets provider addon, the necessary `ClusterRole` and `ClusterRoleBinding` for syncing secrets are automatically created and managed by AKS. In this case, you do not need to manually grant these permissions or create additional RBAC resources for the CSI driver to function.

---

## 3. Example Workflow Diagram

```
[OneCert] → [Azure Key Vault] → [CSI Driver] → [Kubernetes Secret] → [Pod/Ingress]
```

---

## 4. Summary Table

| Step                | Component         | Purpose/Usage                                                                  |
|---------------------|-------------------|--------------------------------------------------------------------------------|
| 1. Registration     | OneCert           | Declare which service tree id/team owns which domain name                      |
| 2. Creation         | Azure Key Vault   | Create a certificate with renew policy from OneCert and store it in Key Vault  |
| 3. Sync             | CSI Driver        | Sync cert from Key Vault to Kubernetes via a k8s SecretProviderClass resource  |
| 4. Consumption      | Pod/Ingress       | Geneva logs: mount as a volume in mdsd container by referencing the SecretProviderClass resource directly<br>UserRP: use the secret in Ingress  |

---

## 5. Related Documentation Links

- [OneCert Portal](https://aka.ms/onecert/)
- [OneCert Usage Guide](https://eng.ms/docs/products/onecert-certificates-key-vault-and-dsms/key-vault-dsms/onecert/docs)
- [Azure Key Vault Certificates](https://learn.microsoft.com/azure/key-vault/certificates/about-certificates)
- [Configure Key Vault integration with AKS](https://learn.microsoft.com/azure/aks/csi-secrets-store-driver)
- [Secrets Store CSI Driver for Kubernetes](https://secrets-store-csi-driver.sigs.k8s.io/)
- [Using Azure Key Vault Provider](https://azure.github.io/secrets-store-csi-driver-provider-azure/docs/getting-started/usage/#set-up-rbac)
- [Kubernetes Ingress TLS](https://kubernetes.io/docs/concepts/services-networking/ingress/#tls)

