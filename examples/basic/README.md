# Basic
In this example there are three Namespaces, ns1, ns2, and  ns3. ns1 contains a Secret and an Intent for that Secret, the Intent only allows Requests from the Namespace ns2. ns2 contains a request for the Intent in ns1, so does ns3. The Request in ns3 will not succeed as it is not allowed by the Intent.
