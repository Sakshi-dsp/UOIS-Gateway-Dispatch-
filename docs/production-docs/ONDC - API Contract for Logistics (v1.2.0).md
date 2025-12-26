
**Version History**

| Version | Date | Changes |
| :---- | :---- | :---- |
|   1.2 | 15th May 2024 | Added 2 new fulfillment states "At-pickup", "At-delivery", details [here](#fulfillment-states-&-mapping-to-order-states); |
|  | 7th Apr 2024 | Removed "Backward Compatibility" section; |
|  | 23rd Dec 2023 | Linking cancellation with IGM, pre-cancel state; |
|  | 18th Nov 2023 | Added support for reverse qc codification; Added support for communication of fulfillment delay (pickup / delivery); |
|  | 4th Nov 2023 | Added tracking attributes in /on\_status to optimize track calls for hyperlocal / hub-based tracking; |
|  | 2nd Nov 2023 | Updated [backward compatibility requirements](#heading=h.b3q3q58cw1mb), for pre-order & post-order flows, so that correct version is used; |
|  | 7th Oct 2023 | Updated [flow](#rules-for-order-confirmation) for /confirm & /on\_confirm to address dangling orders; Additions to /on\_track to support non-hyperlocal orders (in addition to tracking of hyperlocal orders already enabled);  |
|  | 21st Sep 2023 | Added publishing static terms for [transaction level contract](#transaction-level-contract) in /search; Added [backward compatibility requirements](#heading=h.b3q3q58cw1mb); |
|  | 19th July 2023  | Changed /track & /on\_track \- removed callback\_url, updates will be through polling only; Updated swagger link; |
|  | 20th June 2023 | Added to /on\_update \- proof of weight differential, separated updated dimensions & weight (in case of weight differential) from original dimensions & weight; |
|  | 8th June 2023 | Added features as per the feature doc (changes from v1.1 highlighted) |

1. ## Introduction

ONDC API contracts map business requirements for various use case scenarios to a set of attribute keys in different APIs and ensures interoperability, between a buyer NP & seller NP, as follows:

* by defining the set of attribute keys that need to be exchanged to establish a handshake between the buyer NP & seller NP;  
* clearly defining the single source of truth for each attribute key and thereby establishing mutability & immutability of these keys on behalf of the participants at each end of the transaction;

This API contract document is structured as follows:

* JSON payload for each API request & response;  
* Definition & enumerations of specific attribute keys, as applicable;  
* Examples of usage of specific attribute keys for a business requirement, as applicable;  
* Expected behaviour of APIs on the network;

Logistics API contract, defines the interaction between the logistics buyer NP (currently, same as retail seller NP which fills the order) and the logistics seller NP (LSP) which fulfills the order, and includes the following APIs:

1. **Pre-order APIs**  
   * **/search** \- buyer NP specifies the search intent;  
   * **/on\_search** \- seller NP responds with the catalog based on the search intent;  
   * **/init** & **/on\_init** \- buyer & seller NP specify & agree to the terms & conditions prior to placing the order;  
   * **/confirm** \- buyer NP places the order on behalf of the buyer;  
   * **/on\_confirm** \- seller NP responds to the order placed either through auto-acceptance or deferred acceptance or rejection of the order;

2. **Post-order APIs**  
   * **/status** \- buyer NP requests for the current status of the order;  
   * **/on\_status** \- seller NP provides the current status of the order;  
   * **/cancel** \- buyer NP places cancellation request for the order;  
   * **/on\_cancel** \- seller NP responds to the buyer NP cancellation request or cancels the order directly;  
   * **/update** \- buyer NP updates fulfillment details, such as PCC / DCC / authorization codes / payload details;  
   * **/on\_update** \- seller NP responds to the buyer NP request and provides information related to pickup / delivery time slots, shipping label / AWB no / EBN;  
   * **/track** \- buyer NP requests for live tracking of order;  
   * **/on\_track** \- seller NP response with URL for live tracking of order;

All attribute keys, defined in this API contract, are mandatory, unless explicitly mentioned otherwise. Participants may validate the request & response payloads, based on mandatory attribute keys, as defined in this contract and respond with "NACK" (along with appropriate error code as defined [here](https://github.com/ONDC-Official/developer-docs/blob/main/protocol-network-extension/error-codes.md)) for invalid payloads. Any payload, without optional attribute keys, cannot be considered as invalid.

All participants are required to ensure full compliance with this API contract on the live ONDC network.

2. ## Signing & Verification of requests & responses

1. Key pairs, for signing & encryption, can be generated using standard libraries such as [libsodium](https://libsodium.gitbook.io/doc/bindings_for_other_languages).

2. Creating key pairs  
* Create key pairs, for signing (ed25519) & encryption (X25519);  
* Update base64 encoded public keys in registry;  
* Utility to generate signing key pairs and test signing & verification is [here](https://github.com/ONDC-Official/reference-implementations/tree/main/utilities/signing_and_verification);

3. Auth Header Signing  
* Generate UTF-8 byte array from json payload;  
* Generate Blake2b hash from UTF-8 byte array;  
* Create base64 encoding of Blake2b hash, this becomes the digest for signing;  
* Sign the request, using your private signing key, and add the signature to the request authorization header, following steps documented here;

4. Auth Header Verification  
* Extract the digest from the encoded signature in the request;  
* Get the signing\_public\_key from registry using lookup (by using the ukId in the authorization header);  
* Create (UTF-8) byte array from the **raw payload** and generate Blake2b hash;  
* Compare generated Blake2b hash with the decoded digest from the signature in the request;  
* In case of failure to verify, HTTP error 401 should be thrown;

3. ## Registry lookup for verifying requests

1. There are 2 possible participant types, for logistics domain, in the ONDC registry:  
   * Buyer NP  
   * Seller NP

2. Each participant type, defined above, will have an entry in the registry for their entity. Every participant on the ONDC registry will have 1 or more entries in the registry, with "ukid" for each such entry;

3. Signing & Verification of requests and responses using auth headers is defined [here](https://docs.google.com/document/d/1-xECuAHxzpfF8FEZw9iN3vT7D3i6yDDB1u2dEApAjPA/edit);

4. All requests & responses have to be signed by the sender and verified by the receiver. If any request fails verification, it should be rejected (NACK) with http 401 error code;

5. The identity of the sender of a request is sent, through the auth header, as follows:

   keyId="{subscriber\_id}|{unique\_key\_id}|{algorithm}"

6. Signing by Buyer App & Seller App will be using their private key, for which the corresponding public key will be in the registry for their "ukid";

7. Verification of requests using registry /lookup uses [this](https://app.swaggerhub.com/apis/ONDC/ONDC-Registry-Onboarding/2.0.5) API spec. **Participants can either use the registry lookup endpoints from [here](https://github.com/ONDC-Official/developer-docs) or cache the registry locally (with refresh at regular interval) for a local lookup.** Following options are available:

   * **/lookup**

     request:

         {

             "subscriber\_id":"lsp.com",

             "domain":"nic2004:60232",

             "ukId":"UKID1",

             **"country[^1]":"IND",**

             **"city[^2]":"std:080",**

             **"type[^3]":"BPP"**

         }

     response:

     (will be an array with 1 or more objects, depending on whether the entity, identified by the subscriber\_id, has registered as different participants in a domain, e.g. buyer app, seller app)

     \[

         {

             "subscriber\_id":"logistics\_buyer.com",

             "status":"SUBSCRIBED",

             "ukId":"UKID1",

             "subscriber\_url":"https://logistics\_buyer.com/ondc",

             "country":"IND",

             "domain":"nic2004:60232",

             "valid\_from":"2023-05-23T00:00:00.000Z",

             "valid\_until":"2027-05-23T00:00:00.000Z",

             "type":"BPP",

             "signing\_public\_key": "9V6WbbMwWy953zUIIICOOQq4nk8zHHJRhrZ19juApL4=",

             "encr\_public\_key": "MCowBQYDK2VuAyEATDpic13936lmOrDdzMpQox0KWXMwb9sqsdd6fcD1LHM=",

             "created":"2023-05-23T00:00:00.000Z",

             "updated":"2023-05-23T00:00:00.000Z",

             "br\_id":"br\_id1",

             "city":"std:080"

         }

     \]

   * **/vlookup**

     {

       "sender\_subscriber\_id[^4]":"logistics\_buyer.com",

       "request\_id[^5]":"123456789",

       "timestamp[^6]":"2023-01-26T19:00:00.000Z",

       "search\_parameters":

       {

         "domain":"nic2004:60232",

         "subscriber\_id[^7]":"lsp.com",

         "country":"IND",

         "type":"sellerApp[^8]",

         "city":"std:080"

       },

       "signature[^9]":""

     }

   


4. ## Construct of APIs & transaction trail

1. All APIs follow the same construct and include:  
* **Context** \- includes transaction\_id, message\_id and other attribute keys that identify the source & destination of the message being sent through the API request or response;  
* **Message** \- identifies the content of the message e.g. the search intent, catalog details, order details, etc.;

2. All APIs are asynchronous, i.e.:  
* request is sent from sender to recipient;  
* recipient acknowledges the request by sending an "ACK" or sends "NACK" if the recipient cannot validate the request;  
* recipient sends response through the callback for the corresponding request API e.g. /on\_search callback for /search request;  
* recipient of callback (could be the original sender) acknowledges the response by sending an "ACK" or "NACK" if the recipient cannot validate the request;

3. Every request & response pair is identified by a transaction\_id \+ unique message\_id & the sender of a request is able to correlate the response to the request using the above;

4. If a stale request or response is received, i.e. with timestamp earlier than similar request or response (identified by the transaction\_id \+ message\_id) that has already been processed, the recipient can send "NACK" with the error code "65003";

5. Error block can be sent in sync (in response to request) or async (along with callback) mode and should have the following attribute keys: error.type, error.code;

6. ONDC standard error codes are defined [here](https://github.com/ONDC-Official/developer-docs/blob/main/protocol-network-extension/error-codes.md);

7. Transaction trail for a buyer is maintained through a unique transaction\_id for the pre-order stage, i.e. /search, /select, /init, /confirm;

8. Unique id for an order on a network will be identified by the transaction\_id & order id;

9. Swagger link is [here](https://app.swaggerhub.com/apis/ONDC/ONDC-Protocol-Logistics/1.0.25#/);

5. ## API Contract

   ### /search

* Following enhancements are included:

  * **Order preparation time**

    * Logistics buyer NP can send order preparation time in /search, which can be used by LSP to optimize scheduling of their task allocation algorithm;  
    * Order preparation time \= max(time to ship as per catalog, time when RTS expected to be sent);

{  
  "context":  
  {  
    "domain":"nic2004:60232",  
    "country":"IND",  
    "city":"std:080[^10]",  
    "action":"search",  
    "core\_version":"**1.2.0**",  
    "bap\_id":"logistics\_buyer.com",  
    "bap\_uri":"https://logistics\_buyer.com/ondc",  
    "transaction\_id":"T1",  
    "message\_id":"M1",  
    "timestamp":"2023-06-06T21:00:00.000Z",  
    "ttl":"PT30S"  
  },  
  "message":  
  {  
    "intent":  
    {  
      "category":  
      {  
        "id":"Immediate Delivery[^11]"  
      },  
      "provider":  
      {  
        "time[^12]":  
        {  
          "days":"1,2,3,4,5,6,7[^13]",  
          "schedule":  
          {  
            "holidays[^14]":  
            \[  
              "2023-06-29",  
              "2023-08-15"  
            \]  
          },  
          **"duration[^15]":"PT30M",**  
          "range[^16]":  
          {  
            "start":"1100",  
            "end":"2100"  
          }  
        }  
      },  
      "fulfillment":  
      {  
        "type":"**Delivery**[^17]",  
        "start":  
        {  
          "location":  
          {  
            "gps":"12.453544,77.928379[^18]",  
            "address":  
            {  
              "area\_code":"560041"  
            }  
          },  
          **"authorization[^19]":**  
          **{**  
            **"type":"OTP[^20]"**  
          **}**  
        },  
        "end":  
        {  
          "location":  
          {  
            "gps":"12.453544,77.928379[^21]",  
            "address":  
            {  
              "area\_code":"560001"  
            }  
          },  
          **"authorization[^22]":**  
          **{**  
            **"type":"OTP[^23]"**  
          **}**  
        }  
      },  
      "payment":  
      {  
        **"type":"POST-FULFILLMENT[^24]",**  
        "@ondc/org/collection\_amount[^25]":"300.00"  
      },  
      "@ondc/org/payload\_details":  
      {  
        "weight":  
        {  
          "unit":"**kilogram**[^26]",  
          "value":1  
        },  
        "dimensions[^27]":  
        {  
          "length":  
          {  
            "unit":"centimeter[^28]",  
            "value":1  
          },  
          "breadth":  
          {  
            "unit":"centimeter",  
            "value":1  
          },  
          "height":  
          {  
            "unit":"centimeter",  
            "value":1  
          }  
        },  
        "category[^29]":"Grocery",  
        "value":  
        {  
          "currency":"INR",  
          "value":"300.00"  
        },  
        "**dangerous\_goods**[^30]":false  
      }  
    }  
  }  
}

**N.B.**

1. LSP does a serviceability check based on the fulfillment start & end locations and provides a quote for all fulfillment options that meet the search intent;

2. LSP should support search for parent categories such as "Standard Delivery", "Express Delivery" or child categories such as "Immediate Delivery", "Same Day Delivery", "Next Day Delivery". Usage of these categories is defined above;

3. LSP should return all catalog options for "Standard Delivery" including for child categories such as "Immediate Delivery", "Same Day Delivery", "Next Day Delivery". LSP should also return all catalog options for "Express Delivery";

### /on\_search

* Following enhancements are included:

  * **Motorable distance, average time to pickup (useful for Immediate Delivery)**

    * LSP can provide motorable distance (preferably **OSRM**), average time to pickup as part of the catalog response;  
    * Motorable distance will enable logistics buyer NP to optimize their rate card for disconnected logistics;  
    * Average time to pickup (1st mile) will enable logistics buyer NP to correctly estimate their O2D TAT for the buyer;

  * **Catalog response should be sent directly to logistics buyer NP;**

{  
  "context":  
  {  
    "domain":"nic2004:60232",  
    "country":"IND",  
    "city":"std:080",  
    "action":"on\_search",  
    "core\_version":"**1.2.0**",  
    "bap\_id":"logistics\_buyer.com",  
    "bap\_uri":"https://logistics\_buyer.com/ondc",  
    "bpp\_id":"lsp.com",  
    "bpp\_uri":"https://lsp.com/ondc",  
    "transaction\_id":"T1",  
    "message\_id":"M1",  
    "timestamp":"2023-06-06T21:00:30.000Z"  
  },  
  "message":  
  {  
    "catalog":  
    {  
      "bpp/descriptor":  
      {  
        "name[^31]":"LSP Aggregator Inc",  
        **"tags":**  
        **\[**  
          **{**  
            **"code":"bpp\_terms",**  
            **"list":**  
            **\[**  
              **{**  
                **"code":"static\_terms",**  
                **"value":""**  
              **},**  
              **{**  
                **"code":"static\_terms\_new",**  
                **"value":"https://github.com/ONDC-Official/NP-Static-Terms/lspNP\_LSP/1.0/tc.pdf"**  
              **},**  
              **{**  
                **"code":"effective\_date",**  
                **"value":"2023-10-01T00:00:00.000Z"**  
              **}**  
            **\]**  
          **}**  
        **\]**  
      },  
      "bpp/providers":  
      \[  
        {  
          "id":"P1",  
          "descriptor":  
          {  
            "name":"LSP Courier Inc",  
            "short\_desc":"LSP Courier Inc",  
            "long\_desc":"LSP Courier Inc"  
          },  
          "categories":  
          \[  
            {  
              "id":"Immediate Delivery",  
              "time[^32]":  
              {  
                "label":"TAT",  
                "duration":"PT60M",  
                **"timestamp[^33]":"2023-06-06"**  
              }  
            }  
          \],  
          **"fulfillments":**  
          **\[**  
            **{**  
              **"id":"1",**  
              **"type":"Delivery",**  
              **"start":**  
              **{**  
                **"time":**  
                **{**  
                  **"duration[^34]":"PT15M"**  
                **}**  
              **},**  
              **"tags":**  
              **\[**  
                **{**  
                  **"code":"distance[^35]",**  
                  **"list":**  
                  **\[**  
                    **{**  
                      **"code":"motorable\_distance\_type",**  
                      **"value":"kilometer[^36]"**  
                    **},**  
                    **{**  
                      **"code":"motorable\_distance[^37]",**  
                      **"value":"1.8"**  
                    **}**  
                  **\]**  
                **}**  
              **\]**  
            **},**  
            **{**  
              **"id":"2",**  
              **"type":"RTO"**  
            **}**  
          **\],**  
          "locations[^38]":  
          \[  
            {  
              "id":"L1",  
              "gps":"12.967555,77.749666",  
              "address":  
              {  
                "street":"Jayanagar 4th Block",  
                "city":"Bengaluru",  
                "area\_code":"560076",  
                "state":"KA"  
              }  
            }  
          \],  
          "items":  
          \[  
            {  
              "id":"I1",  
              "parent\_item\_id":"",  
              "category\_id":"Immediate Delivery",  
              "fulfillment\_id":"1[^39]",  
              "descriptor":  
              {  
                **"code":"P2P[^40]",**  
                "name":"60 min delivery",  
                "short\_desc":"60 min delivery for F\&B",  
                "long\_desc":"60 min delivery for F\&B"  
              },  
              "price":  
              {  
                "currency":"INR",  
                "value":"59.00[^41]"  
              },  
              "time[^42]":  
              {  
                "label":"TAT",  
                "duration":"PT45M",  
                "timestamp[^43]":"2023-06-06"  
              }  
            },  
            {  
              "id":"I2",  
              "parent\_item\_id[^44]":"I1",  
              "category\_id":"Immediate Delivery",  
              "fulfillment\_id":"2[^45]",  
              "descriptor":  
              {  
                **"code":"P2P",**  
                "name":"RTO quote",  
                "short\_desc":"RTO quote",  
                "long\_desc":"RTO quote"  
              },  
              "price":  
              {  
                "currency":"INR",  
                "value":"23.60[^46]"  
              },  
              "time[^47]":  
              {  
                "label":"TAT",  
                "duration":"PT60M",  
                "timestamp[^48]":"2023-06-06"  
              }  
            }  
          \]  
        }  
      \]  
    }  
  }  
}

**Example**

1. Logistics buyer NP can send /search with category "Standard Delivery" or "Express Delivery" and the LSP is required to respond with all appropriate options e.g. for "Standard Delivery", the LSP responds with options for "Immediate Delivery", "Same Day Delivery", "Next Day Delivery", as applicable;  
2. Logistics buyer NP can also send /search for specific category such as "Immediate Delivery", "Same Day Delivery", "Next Day Delivery" and the LSP either responds with options available or doesn’t respond;  
3. Catalog options provided by LSP can be either "P2P" (point-to-point) or "P2H2P" (point-to-hub-to-point);  
4. For "P2P", the rider delivers the package directly from the pickup location (merchant’s) to the drop-off location (buyer’s), for "P2H2P", the package is routed through a hub;  
5. Logistics buyer NP can inform the merchants of the type of packaging required based on the catalog option i.e. "P2P" or "P2H2P". Since "P2H2P" fulfillments are routed through a hub, AWB no is required;  
6. For different logistics categories, following validations should be in place:  
   1. "Immediate Delivery" \- time.duration \<= PT60M for the same day (time.timestamp is same day as Context.timestamp);  
   2. "Same Day Delivery" \- for the same day (time.timestamp is same day as Context.timestamp);  
   3. "Next Day Delivery" \- for the next day (time.timestamp is the day after the day for Context.timestamp)

##### /on\_search sync response (ACK)

In the sync response to /on\_search, the logistics buyer NP can provide the following information:

* Acceptance or rejection of LSP static terms

  ###### */on\_search (ACK response)*

"response":  
{  
  "context":  
  {  
    "domain":"nic2004:60232",  
    "country":"IND",  
    "city":"std:080",  
    "action":"on\_search",  
    "core\_version":"**1.2.0**",  
    "bap\_id":"logistics\_buyer.com",  
    "bap\_uri":"https://logistics\_buyer.com/ondc",  
    "bpp\_id":"lsp.com",  
    "bpp\_uri":"https://lsp.com/ondc",  
    "transaction\_id":"T1",  
    "message\_id":"M1",  
    "timestamp":"2023-06-06T21:00:30.000Z"  
  },  
  "message":  
  {  
    "ack":  
    {  
      "status":"ACK",  
      **"tags":**  
      **\[**  
        **{**  
          **"code":"bap\_terms",**  
          **"list":**  
          **\[**  
            **{**  
              **"code":"accept\_bpp\_terms",**  
              **"value":"Y[^49]"**  
            **}**  
          **\]**  
        **}**  
      **\]**  
    }  
  }  
}

### /init

{  
  "context":  
  {  
    "domain":"nic2004:60232",  
    "country":"IND",  
    "city":"std:080",  
    "action":"init",  
    "core\_version":"**1.2.0**",  
    "bap\_id":"logistics\_buyer.com",  
    "bap\_uri":"https://logistics\_buyer.com/ondc",  
    "bpp\_id":"lsp.com",  
    "bpp\_uri":"https://lsp.com/ondc",  
    "transaction\_id":"T1",  
    "message\_id":"M2",  
    "timestamp":"2023-06-06T21:30:00.000Z",  
    "ttl":"PT30S"  
  },  
  "message":  
  {  
    "order":   
    {  
      "provider":  
      {  
        "id":"P1",  
        "locations[^50]":  
        \[  
          {  
            "id":"L1"  
          }  
        \]  
      },  
      "items":  
      \[  
        {  
          "id":"I1",  
          **"fulfillment\_id":"1",**  
          "category\_id":"Immediate Delivery",  
          **"descriptor":**  
          **{**  
             **"code":"P2P"**  
          **}**  
        }  
      \],  
      "fulfillments":  
      \[  
        {  
          "id":"1[^51]",  
          "type":"**Delivery**",  
          "start":  
          {  
            "location":  
            {  
              "gps":"12.453544,77.928379",  
              "address[^52]":  
              {  
                "name":"My store name",  
                "building":"My building name",  
                "locality":"My street name",  
                "city":"Bengaluru",  
                "state":"Karnataka",  
                "country":"India",  
                "area\_code":"560041"  
              }  
            },  
            "contact":  
            {  
              "phone":"9886098860",  
              "email":"abcd.efgh@gmail.com"  
            }  
          },  
          "end":  
          {  
            "location":  
            {  
              "gps":"12.453544,77.928379",  
              "address[^53]":  
              {  
                "name":"My house \#",  
                "building":"My house or building name",  
                "locality":"My street name",  
                "city":"Bengaluru",  
                "state":"Karnataka",  
                "country":"India",  
                "area\_code":"560076"  
              }  
            },  
            "contact":  
            {  
              "phone":"9886098860",  
              "email[^54]":"abcd.efgh@gmail.com"  
            }  
          }  
        }  
      \],  
      "billing[^55]":  
      {  
        "name[^56]":"ONDC Logistics Buyer NP",  
        "address[^57]":  
        {  
          "name":"My house or building no",  
          "building":"My house or building name",  
          "locality":"Jayanagar",  
          "city":"Bengaluru",  
          "state":"Karnataka",  
          "country":"India",  
          "area\_code":"560076"  
        },  
        "**tax\_number**[^58]":"XXXXXXXXXXXXXXX",  
        "phone":"9886098860",  
        "**email**[^59]":"abcd.efgh@gmail.com",  
        "created\_at":"2023-02-06T21:30:00.000Z",  
        "updated\_at":"2023-02-06T21:30:00.000Z"  
      },  
      "payment[^60]":  
      {  
        **"@ondc/org/collection\_amount[^61]":"300.00",**  
        **"collected\_by[^62]":"BPP",**  
        "type":"ON-FULFILLMENT",  
        "@ondc/org/settlement\_details[^63]":  
        \[  
          {  
            "settlement\_counterparty":"buyer-app",  
            "settlement\_type":"upi[^64]",  
            "beneficiary\_name":"xxxxx",  
            "upi\_address":"gft@oksbi",  
            "settlement\_bank\_account\_no":"XXXXXXXXXX",  
            "settlement\_ifsc\_code":"XXXXXXXXX"  
          }  
        \]  
      }  
    }  
  }  
}

**N.B.**

1. LSP is required to perform serviceability check while processing /init and return appropriate [error codes](https://github.com/ONDC-Official/developer-docs/blob/main/protocol-network-extension/error-codes.md) if the pickup and / or dropoff locations aren’t serviceable;

   ### /on\_init

* Following enhancements are included:

  * **Cancellation terms**

    * Cancellation terms will be defined by the LSP for the order in /on\_init;  
    * Cancellation terms will be defined based on the fulfillment state & reason codes and include cancellation fees (if any);  
    * If LSP cancellation terms are not acceptable to Logistics Buyer NP, they should NACK the /on\_init with error code 62505;

  * **Itemizing tax in quote**

    * Currently, fulfillment charges provided by LSP are tax inclusive;  
    * With this change, fulfillment charges will have separate quote line item for taxes;  
    * Logistics buyer NP can forward the itemized fulfillment charges to retail buyer NP by consolidating the tax line items for other fulfillment charges;

  * **Inline check for rider availability (hyperlocal only)**

    * LSP can provide information on whether inline check for rider availability was done;

  * **Change in quote title type**

    * Change in quote title type to align with retail spec; updated enum & mapping is defined below:

| Current | New | Meaning |
| :---- | :---- | :---- |
| Delivery Charge | delivery | Delivery charge |
| RTO Charge | rto | RTO charge |
| Reverse QC Charge | reverseqc | Reverse QC charge |
| Tax | tax | Tax on delivery / rto / reverseqc |
|  | **diff** | **Additional delivery charge due to weight differential** |
|  | **tax\_diff** | **Tax on additional delivery charge due to weight differential** |
|  | **discount** | **Discount on delivery charges (if any)** |

  * Configurable terms for [transaction level contract](#transaction-level-contract);

{  
  "context":  
  {  
    "domain":"nic2004:60232",  
    "country":"IND",  
    "city":"std:080",  
    "action":"on\_init",  
    "core\_version":"**1.2.0**",  
    "bap\_id":"logistics\_buyer.com",  
    "bap\_uri":"https://logistics\_buyer.com/ondc",  
    "bpp\_id":"lsp.com",  
    "bpp\_uri": "https://lsp.com/ondc",  
    "transaction\_id":"T1",  
    "message\_id":"M2",  
    "timestamp":"2023-02-06T21:30:30.000Z"  
  },  
  "message":  
  {  
    "order":   
    {  
      "provider":  
      {  
        "id":"P1",  
        "locations[^65]":  
        \[  
          {  
            "id":"L1"  
          }  
        \]  
      },  
      "items":  
      \[  
        {  
          "id":"I1",  
          **"fulfillment\_id":"1"**  
        }  
      \],  
      **"fulfillments":**  
      **\[**  
        **{**  
          **"id":"1",**  
          **"type":"Delivery",**  
          **"start":**  
          **{**  
            **"location":**  
            **{**  
              **"gps":"12.4535445,77.9283792",**  
              **"address":**  
              **{**  
                **"name":"Store name",**  
                **"building":"House or building name",**  
                **"locality":"Locality",**  
                **"city":"Bengaluru",**  
                **"state":"Karnataka",**  
                **"country":"India",**  
                **"area\_code":"560041"**  
              **}**  
            **},**  
            **"contact":**  
            **{**  
              **"phone":"9886098860",**  
              **"email":"abcd.efgh@gmail.com"**  
            **}**  
          **},**  
          **"end":**  
          **{**  
            **"location":**  
            **{**  
              **"gps":"12.453544,77.928379",**  
              **"address":**  
              **{**  
                **"name":"My house \#",**  
                **"building":"My house or building name",**  
                **"locality":"locality",**  
                **"city":"Bengaluru",**  
                **"state":"Karnataka",**  
                **"country":"India",**  
                **"area\_code":"560076"**  
              **}**  
            **},**  
            **"contact":**  
            **{**  
              **"phone":"9886098860",**  
              **"email":"abcd.efgh@gmail.com"**  
            **}**  
          **},**  
          **"tags":**  
          **\[**  
            **{**  
              **"code":"rider\_check[^66]",**  
              **"list":**  
              **\[**  
                **{**  
                  **"code":"inline\_check\_for\_rider",**  
                  **"value":"yes"**  
                **}**  
              **\]**  
            **}**  
          **\]**  
        **}**  
      **\],**  
      "quote":  
      {  
        "price":  
        {  
          "currency":"INR",  
          "value":"59.00"  
        },  
        "breakup":  
        \[  
          **{**  
            **"@ondc/org/item\_id":"I1",**  
            **"@ondc/org/title\_type":"delivery[^67]",**  
            **"price":**  
            **{**  
              **"currency":"INR",**  
              **"value":"50.00"**  
            **}**  
          **},**  
          **{**  
            **"@ondc/org/item\_id":"I1",**  
            **"@ondc/org/title\_type":"tax",**  
            **"price":**  
            **{**  
              **"currency":"INR",**  
              **"value":"9.00"**  
            **}**  
          **}**  
        \],  
        **"ttl":"PT15M[^68]"**  
      },  
      "payment":  
      {  
        **"@ondc/org/collection\_amount[^69]":"300.00",**  
        "type":"ON-FULFILLMENT[^70]",  
        "collected\_by":"BPP",  
        "@ondc/org/settlement\_details[^71]":  
        \[  
          {  
            "settlement\_counterparty":"buyer-app",  
            "settlement\_type": "upi",  
            "beneficiary\_name":"xxxxx",  
            "upi\_address":"gft@oksbi",  
            "settlement\_bank\_account\_no":"XXXXXXXXXX",  
            "settlement\_ifsc\_code":"XXXXXXXXX"  
          }  
        \]  
      },  
      **"cancellation\_terms":**  
      **\[**  
        **{**  
          **"fulfillment\_state":**  
          **{**  
            **"descriptor":**  
            **{**  
              **"code":"Pending[^72]",**  
              **"short\_desc[^73]":"008"**  
            **}**  
          **},**  
          **"cancellation\_fee[^74]":**  
          **{**  
            **"percentage":"0.00",**  
            **"amount":**  
            **{**  
              **"currency":"INR",**  
              **"value":"0.00"**  
            **}**  
          **}**  
        **},**  
        **{**  
          **"fulfillment\_state":**  
          **{**  
            **"descriptor":**  
            **{**  
              **"code":"Agent-assigned",**  
              **"short\_desc":"001,003"**  
            **}**  
          **},**  
          **"cancellation\_fee":**  
          **{**  
            **"percentage":"100.00",**  
            **"amount":**  
            **{**  
              **"currency":"INR",**  
              **"value":"50.00"**  
            **}**  
          **}**  
        **},**  
        **{**  
          **"fulfillment\_state":**  
          **{**  
            **"descriptor":**  
            **{**  
              **"code":"Order-picked-up",**  
              **"short\_desc":"001,003"**  
            **}**  
          **},**  
          **"cancellation\_fee":**  
          **{**  
            **"percentage":"100.00",**  
            **"amount":**  
            **{**  
              **"currency":"INR",**  
              **"value":"50.00"**  
            **}**  
          **}**  
        **},**  
        **{**  
          **"fulfillment\_state":**  
          **{**  
            **"descriptor":**  
            **{**  
              **"code":"Out-for-delivery",**  
              **"short\_desc":"011,012,013,014,015[^75]"**  
            **}**  
          **},**  
          **"cancellation\_fee":**  
          **{**  
            **"percentage":"100.00",**  
            **"amount[^76]":**  
            **{**  
              **"currency":"INR",**  
              **"value":"50.00"**  
            **}**  
          **}**  
        **}**  
      **\],**  
      **"tags":**  
      **\[**  
        **{**  
          **"code":"bpp\_terms[^77]",**  
          **"list":**  
          **\[**  
            **{**  
              **"code":"max\_liability",**  
              **"value":"2"**  
            **},**  
            **{**  
              **"code":"max\_liability\_cap",**  
              **"value":"10000"**  
            **},**  
            **{**  
              **"code":"mandatory\_arbitration",**  
              **"value":"false"**  
            **},**  
            **{**  
              **"code":"court\_jurisdiction",**  
              **"value":"Bengaluru"**  
            **},**  
            **{**  
              **"code":"delay\_interest",**  
              **"value":"1000"**  
            **},**  
            **{**  
              **"code":"static\_terms[^78]",**  
              **"value":"https://github.com/ONDC-Official/protocol-network-extension/discussions/79"**  
            **}**  
          **\]**  
        **}**  
      **\]**  
    }  
  }  
}

**N.B.**

1. LSP is required to perform check for serviceability, for the /init request, and return appropriate error codes along with the response;

2. Order quote should comply with the following:  
   1. Precision for all prices in quote can be maximum of 2 decimal digits;  
   2. Order value, i.e. quote.price.value \= Σ quote line items, i.e. quote.breakup\[\].price.value;  
   3. Quote provided here should match what was provided in /on\_search and is frozen until order /confirm;

   ### 

   ### /confirm

* Following enhancements are included:

  * **Fulfillment state TAT breach**

    * LSP provides a ship-to-delivery (S2D) TAT to the logistics buyer NP, along with a quote;  
    * Retail Seller NP uses this to compute an order-to-delivery (O2D) TAT as follows: ship-to-delivery (S2D) \+ order-to-ship (O2S) \+ average time to pickup (proposed above);  
    * S2D TAT will also be a part of the order in /confirm and /on\_confirm;  
    * When there is an O2D breach, the retail buyer NP / seller NP may initiate cancellation of the retail order / fulfillment which will cascade into cancellation of the logistics order;  
    * Liabilities in case of O2D breach will be defined in the terms & conditions;

  * **RTO flow**

    * Every fulfillment will have an identifier for whether RTO results in actual return to origin;

  * **Codifying reverse QC SOP**

    * Reverse QC SOP is in the form of an online checklist, with or without instructions, and provided by Seller NP;  
    * Logistics agent updates the online checklist with status of inspection of return items;  
    * Sample checklist is included [here](https://docs.google.com/spreadsheets/d/1WLCB-PqaeWBvqCW_mOq22YJHsUXyWWK8/edit#gid=1890861429) and this should preferably include instructions on whether a item (for return) can be picked up or not;

  * **Changing tag structure for RTS**

    * tag structure for ready\_to\_ship will be changed to ensure consistency with the new tag structure;

{  
  "context":  
  {  
    "domain":"nic2004:60232",  
    "country":"IND",  
    "city":"std:080",  
    "action":"confirm",  
    "core\_version":"**1.2.0**",  
    "bap\_id":"logistics\_buyer.com",  
    "bap\_uri":"https://logistics\_buyer.com/ondc",  
    "bpp\_id":"lsp.com",  
    "bpp\_uri":"https://lsp.com/ondc",  
    "transaction\_id":"T1",  
    "message\_id":"M3",  
    "timestamp":"2023-06-06T22:00:00.000Z",  
    "ttl":"PT30S"  
  },  
  "message":  
  {  
    "order":   
    {  
      "id":"O2[^79]",  
      "state":"Created",  
      "provider":  
      {  
        "id":"P1",  
        "locations[^80]":  
        \[  
          {  
            "id":"L1"  
          }  
        \]   
      },  
      "items":  
      \[  
        {  
          "id":"I1",  
          **"fulfillment\_id":"1",**  
          "category\_id":"Immediate Delivery",  
          **"descriptor":**  
          **{**  
             **"code":"P2P"**  
          **},**  
          **"time[^81]":**  
          **{**  
            **"label":"TAT",**  
            **"duration":"PT45M",**  
            **"timestamp":"2023-06-06"**  
          **}**  
        }  
      \],  
      "quote":  
      {  
        "price":  
        {  
          "currency":"INR",  
          "value": "59.00"  
        },  
        "breakup":  
        \[  
          {  
            "@ondc/org/item\_id":"I1",  
            "@ondc/org/title\_type":"**delivery**",  
            "price":  
            {  
              "currency":"INR",  
              "value":"50.00"  
            }  
          },  
          **{**  
            **"@ondc/org/item\_id":"I1",**  
            **"@ondc/org/title\_type":"tax",**  
            **"price":**  
            **{**  
              **"currency":"INR",**  
              **"value":"9.00"**  
            **}**  
          **}**  
        \]  
      },  
      "fulfillments":  
      \[  
        {  
          "id":"1",  
          "type":"**Delivery**",  
          "@ondc/org/awb\_no[^82]":"1227262193237777",  
          "start":  
          {  
            **"time[^83]":**  
            **{**  
              **"duration":"PT15M"**  
            **},**  
            "person":  
            {  
              "name":"person\_name"  
            },  
            "location":  
            {  
              "gps":"12.4535445,77.9283792",  
              "address":  
              {  
                "name":"Store name",  
                "building":"House or building name",  
                "locality":"Locality",  
                "city":"Bengaluru",  
                "state":"Karnataka",  
                "country":"India",  
                "area\_code":"560041"  
              }  
            },  
            "contact":  
            {  
              "phone":"9886098860",  
              "email":"abcd.efgh@gmail.com"  
            },  
            "instructions[^84]":  
            {  
              "code":"2[^85]",  
              "short\_desc":"value of PCC[^86]",  
              "long\_desc":"additional instructions for pickup[^87]",  
              **"additional\_desc[^88]":**  
              **{**  
                **"content\_type":"text/html",**  
                **"url":"https://reverse\_qc\_sop\_form.htm"**  
              **}**  
            }  
          },  
          "end":  
          {  
            "person":  
            {  
              "name":"person\_name"  
            },  
            "location":  
            {  
              "gps":"12.453544,77.928379",  
              "address":  
              {  
                "name":"My house \#",  
                "building":"My house or building name",  
                "locality":"locality",  
                "city":"Bengaluru",  
                "state":"Karnataka",  
                "country":"India",  
                "area\_code":"560076"  
              }  
            },  
            "contact":  
            {  
              "phone":"9886098860",  
              "email":"abcd.efgh@gmail.com"  
            },  
            "instructions[^89]":  
            {  
              "code":"3[^90]",  
              "short\_desc":"value of DCC[^91]",  
              "long\_desc":"additional instructions for delivery[^92]"  
            }  
          },  
          **"tags":**  
          **\[**  
            **{**  
              **"code":"state",**  
              **"list":**  
              **\[**  
                **{**  
                  **"code":"ready\_to\_ship[^93]",**  
                  **"value":"no[^94]"**  
                **}**  
              **\]**  
            **},**  
            **{**  
              **"code":"rto\_action",**  
              **"list":**  
              **\[**  
                **{**  
                  **"code":"return\_to\_origin[^95]",**  
                  **"value":"no"**  
                **}**  
              **\]**  
            **},**  
            **{**  
              **"code":"reverseqc\_input[^96]",**  
              **"list":**  
              **\[**  
                **{**  
                  **"code":"P001",**  
                  **"value":"Atta"**  
                **},**  
                **{**  
                  **"code":"P003",**  
                  **"value":"1"**  
                **},**  
                **{**  
                  **"code":"Q001",**  
                  **"value":""**  
                **}**  
              **\]**  
            **}**  
          **\]**  
        }  
      \],  
      "billing":  
      {  
        "name":"ONDC Logistics Buyer NP",  
        "address":  
        {  
          "name":"name",  
          **"building":"My house or building name",**  
          "locality":"My street name",  
          "city":"Bengaluru",  
          "state":"Karnataka",  
          "country":"India",  
          "area\_code":"560076"  
        },  
        "tax\_number":"29AAACU1901H1ZK",  
        "phone":"9886098860",  
        "email":"abcd.efgh@gmail.com",  
        **"created\_at":"2023-06-06T21:30:00.000Z",**  
        **"updated\_at":"2023-06-06T21:30:00.000Z"**  
      },  
      "payment":  
      {  
        "@ondc/org/collection\_amount":"300.00",  
        **"collected\_by":"BPP",**  
        "type":"ON-FULFILLMENT",  
        "@ondc/org/settlement\_details[^97]":  
        \[  
          {  
            "settlement\_counterparty":"buyer-app",  
            "settlement\_type":"upi",  
            "upi\_address":"gft@oksbi",  
            "settlement\_bank\_account\_no":"XXXXXXXXXX",  
            "settlement\_ifsc\_code":"XXXXXXXXX"  
          }  
        \]  
      },  
      "@ondc/org/linked\_order":  
      {  
        "items":  
        \[  
          {  
            "category\_id":"Grocery[^98]",  
            "descriptor":  
            {  
              "name":"Atta"  
            },  
            "quantity":  
            {  
              "count":2,  
              "measure":  
              {  
                "unit":"**kilogram[^99]**",  
                "value":0.5  
              }  
            },  
            "price":  
            {  
              "currency":"INR",  
              "value":"150.00"  
            }      
          }  
        \],  
        "provider":  
        {  
          "descriptor":  
          {  
            "name":"Aadishwar Store"  
          },  
          "address":  
          {  
            "name":"KHB Towers",  
            "building":"Building or House No",  
            "locality":"Koramangala",  
            "city":"Bengaluru",  
            "state":"Karnataka",  
            "area\_code":"560070"  
          }  
        },  
        "order":  
        {  
          "id":"O1",  
          "weight":  
          {  
            "unit":"**kilogram[^100]**",  
            "value":1  
          },  
          **"dimensions[^101]":**  
          **{**  
            **"length":**  
            **{**  
              **"unit":"centimeter[^102]",**  
              **"value":1**  
            **},**  
            **"breadth":**  
            **{**  
              **"unit":"centimeter",**  
              **"value":1**  
            **},**  
            **"height":**  
            **{**  
              **"unit":"centimeter",**  
              **"value":1**  
            **}**  
          **}**  
        }  
      },  
      "tags":  
      \[  
        {  
          "code":"bpp\_terms",  
          "list":  
          \[  
            {  
              "code":"max\_liability",  
              "value":"2"  
            },  
            {  
              "code":"max\_liability\_cap",  
              "value":"10000"  
            },  
            {  
              "code":"mandatory\_arbitration",  
              "value":"false"  
            },  
            {  
              "code":"court\_jurisdiction",  
              "value":"Bengaluru"  
            },  
            {  
              "code":"delay\_interest",  
              "value":"1000"  
            },  
            {  
              "code":"static\_terms",  
              "value":"https://github.com/ONDC-Official/protocol-network-extension/discussions/79"  
            }  
          \]  
        },  
        **{**  
          **"code":"bap\_terms",**  
          **"list":**  
          **\[**  
            **{**  
              **"code":"accept\_bpp\_terms[^103]",**  
              **"value":"Y"**  
            **}**  
          **\]**  
        **}**  
      \],  
      "created\_at":"2023-06-06T22:00:00.000Z",  
      "updated\_at":"2023-06-06T22:00:00.000Z"  
    }  
  }  
}

**N.B.**

1. LSP is required to perform check for serviceability, for the /confirm request, and return appropriate error codes along with the response;  
2. LSP creates the shipment manifest on successful processing of /confirm;

   ### /on\_confirm

{  
  "context":  
  {  
    "domain":"nic2004:60232",  
    "country":"IND",  
    "city":"std:080",  
    "action":"on\_confirm",  
    "core\_version":"**1.2.0**",  
    "bap\_id":"logistics\_buyer.com",  
    "bap\_uri":"https://logistics\_buyer.com/ondc",  
    "bpp\_id":"lsp.com",  
    "bpp\_uri":"https://lsp.com/ondc",  
    "transaction\_id":"T1",  
    "message\_id":"M3",  
    "timestamp":"2023-06-06T22:00:30.000Z"  
  },  
  "message":  
  {  
    "order":  
    {  
      "id":"O2",  
      "state":"Accepted[^104]",  
      "provider":  
      {  
        "id":"P1",  
        "locations[^105]":  
        \[  
          {  
            "id":"L1"  
          }  
        \]  
      },  
      "items":  
      \[  
        {  
          "id":"I1",  
          **"fulfillment\_id":"1",**  
          "category\_id":"Same Day Delivery",  
          **"descriptor":**  
          **{**  
             **"code":"P2P"**  
          **},**  
          **"time[^106]":**  
          **{**  
            **"label":"TAT",**  
            **"duration":"PT45M",**  
            **"timestamp":"2023-06-06"**  
          **}**  
        }  
      \],  
      "quote":  
      {  
        "price":  
        {  
          "currency":"INR",  
          "value":"59.00"  
        },  
        "breakup":  
        \[  
          {  
            "@ondc/org/item\_id":"I1",  
            "@ondc/org/title\_type":"**delivery**",  
            "price":  
            {  
              "currency":"INR",  
              "value":"50.00"  
            }  
          },  
          {  
            "@ondc/org/item\_id":"I1",  
            "@ondc/org/title\_type":"**tax**",  
            "price":  
            {  
              "currency":"INR",  
              "value":"9.00"  
            }  
          }  
        \]  
      },  
      "fulfillments":  
      \[  
        {  
          "id":"1",  
          "type":"Delivery",  
          "state":  
          {  
            "descriptor":  
            {  
              "code":"Pending"  
            }  
          },  
          "@ondc/org/awb\_no[^107]":"1227262193237777",  
          "tracking":false[^108],  
          "start[^109]":  
          {  
            **"person":**  
            **{**  
              **"name":"person\_name"**  
            **},**  
            **"location":**  
            **{**  
              **"gps":"12.453544,77.928379",**  
              **"address":**  
              **{**  
                **"name":"Store name",**  
                **"building":"My house or building name",**  
                **"locality":"My street name",**  
                **"city":"Bengaluru",**  
                **"state":"Karnataka",**  
                **"country":"India",**  
                **"area\_code":"560041"**  
              **}**  
            **},**  
            **"contact":**  
            **{**  
              **"phone":"9886098860",**  
              **"email":"abcd.efgh@gmail.com"**  
            **},**  
            **"instructions":**  
            **{**  
              **"code":"2",**  
              **"short\_desc":"value of PCC",**  
              **"long\_desc":"QR code will be attached to package",**  
              **"images":**  
              **\[**  
                **"link to downloadable shipping label (required for P2H2P)[^110]"**  
              **\],**  
              **"additional\_desc[^111]":**  
              **{**  
                **"content\_type":"text/html",**  
                **"url":"https://reverse\_qc\_sop\_form.htm"**  
              **}**  
            **},**  
            "time":  
            {  
              **"duration":"PT15M",**  
              "range[^112]":  
              {  
                "start":"2023-06-06T22:30:00.000Z",  
                "end":"2023-06-06T22:45:00.000Z"  
              }  
            }  
          },  
          "end":  
          {  
            **"person":**  
            **{**  
              **"name":"person\_name"**  
            **},**  
            **"location":**  
            **{**  
              **"gps":"12.4535445,77.9283792",**  
              **"address":**  
              **{**  
                **"name":"My house or building \#",**  
                **"building":"My house or building name",**  
                **"locality":"My street name",**  
                **"city":"Bengaluru",**  
                **"state":"Karnataka",**  
                **"country":"India",**  
                **"area\_code":"560076"**  
              **}**  
            **},**  
            **"contact":**  
            **{**  
              **"phone":"9886098860",**  
              **"email":"abcd.efgh@gmail.com"**  
            **},**  
            **"instructions":**  
            **{**  
              **"code":"3",**  
              **"short\_desc":"value of DCC"**  
            **},**  
            "time":  
            {  
              "range[^113]":  
              {  
                "start":"2023-06-06T23:00:00.000Z",  
                "end":"2023-06-06T23:15:00.000Z"  
              }  
            }  
          },  
          "agent[^114]":  
          {  
            "name":"agent\_name",  
            "phone":"9886098860"  
          },  
          "vehicle[^115]":  
          {  
            "registration":"3LVJ945"  
          },  
          **"tags":**  
          **\[**  
            **{**  
              **"code":"weather\_check[^116]",**  
              **"list":**  
              **\[**  
                **{**  
                  **"code":"raining",**  
                  **"value":"yes"**  
                **}**  
              **\]**  
            **},**  
            **{**  
              **"code":"state",**  
              **"list":**  
              **\[**  
                **{**  
                  **"code":"ready\_to\_ship",**  
                  **"value":"yes"**  
                **}**  
              **\]**  
            **},**  
            **{**  
              **"code":"rto\_action",**  
              **"list":**  
              **\[**  
                **{**  
                  **"code":"return\_to\_origin",**  
                  **"value":"no"**  
                **}**  
              **\]**  
            **},**  
            **{**  
              **"code":"reverseqc\_input[^117]",**  
              **"list":**  
              **\[**  
                **{**  
                  **"code":"P001",**  
                  **"value":"Atta"**  
                **},**  
                **{**  
                  **"code":"P003",**  
                  **"value":"1"**  
                **},**  
                **{**  
                  **"code":"Q001",**  
                  **"value":""**  
                **}**  
              **\]**  
            **}**  
          **\]**  
        }  
      \],  
      "billing[^118]":  
      {  
        "name":"ONDC Logistics Buyer NP",  
        "address":  
        {  
          "name":"My building no",  
          **"building":"My building name",**  
          "locality":"My street name",  
          "city":"Bengaluru",  
          "state":"Karnataka",  
          "country":"India",  
          "area\_code":"560076"  
        },  
        "tax\_number":"XXXXXXXXXXXXXXX",  
        "phone":"9886098860",  
        "email":"abcd.efgh@gmail.com",  
        **"created\_at":"2023-02-06T21:30:00.000Z",**  
        **"updated\_at":"2023-02-06T21:30:00.000Z"**  
      },  
      **"payment[^119]":**  
      **{**  
        **"@ondc/org/collection\_amount":"300.00",**  
        **"collected\_by":"BPP",**  
        **"type":"ON-FULFILLMENT",**  
        **"@ondc/org/settlement\_details[^120]":**  
        **\[**  
          **{**  
            **"settlement\_counterparty":"buyer-app",**  
            **"settlement\_type":"upi",**  
            **"upi\_address":"gft@oksbi",**  
            **"settlement\_bank\_account\_no":"XXXXXXXXXX",**  
            **"settlement\_ifsc\_code":"XXXXXXXXX"**  
          **}**  
        **\]**  
      **},**  
      **"@ondc/org/linked\_order":**  
      **{**  
        **"items":**  
        **\[**  
          **{**  
            **"category\_id":"Grocery[^121]",**  
            **"descriptor":**  
            **{**  
              **"name":"Atta"**  
            **},**  
            **"quantity":**  
            **{**  
              **"count":2,**  
              **"measure":**  
              **{**  
                **"unit":"kilogram[^122]",**  
                **"value":0.5**  
              **}**  
            **},**  
            **"price":**  
            **{**  
              **"currency":"INR",**  
              **"value":"150.00"**  
            **}**      
          **}**  
        **\],**  
        **"provider":**  
        **{**  
          **"descriptor":**  
          **{**  
            **"name":"Aadishwar Store"**  
          **},**  
          **"address":**  
          **{**  
            **"name":"KHB Towers",**  
            **"building":"Building or House No",**  
            **"locality":"Koramangala",**  
            **"city":"Bengaluru",**  
            **"state":"Karnataka",**  
            **"area\_code":"560070"**  
          **}**  
        **},**  
        **"order":**  
        **{**  
          **"id":"O1",**  
          **"weight":**  
          **{**  
            **"unit":"kilogram[^123]",**  
            **"value":1**  
          **},**  
          **"dimensions[^124]":**  
          **{**  
            **"length":**  
            **{**  
              **"unit":"centimeter",**  
              **"value":1**  
            **},**  
            **"breadth":**  
            **{**  
              **"unit":"centimeter",**  
              **"value":1**  
            **},**  
            **"height":**  
            **{**  
              **"unit":"centimeter",**  
              **"value":1**  
            **}**  
          **}**  
        **}**  
      **},**  
      **"cancellation\_terms[^125]":**  
      **\[**  
        **{**  
          **"fulfillment\_state":**  
          **{**  
            **"descriptor":**  
            **{**  
              **"code":"Pending",**  
              **"short\_desc":"008"**  
            **}**  
          **},**  
          **"cancellation\_fee":**  
          **{**  
            **"percentage":"0.00",**  
            **"amount":**  
            **{**  
              **"currency":"INR",**  
              **"value":"0.00"**  
            **}**  
          **}**  
        **},**  
        **{**  
          **"fulfillment\_state":**  
          **{**  
            **"descriptor":**  
            **{**  
              **"code":"Agent-assigned",**  
              **"short\_desc":"001,003"**  
            **}**  
          **},**  
          **"cancellation\_fee":**  
          **{**  
            **"percentage":"100.00",**  
            **"amount":**  
            **{**  
              **"currency":"INR",**  
              **"value":"50.00"**  
            **}**  
          **}**  
        **},**  
        **{**  
          **"fulfillment\_state":**  
          **{**  
            **"descriptor":**  
            **{**  
              **"code":"Order-picked-up",**  
              **"short\_desc":"001,003"**  
            **}**  
          **},**  
          **"cancellation\_fee":**  
          **{**  
            **"percentage":"100.00",**  
            **"amount":**  
            **{**  
              **"currency":"INR",**  
              **"value":"50.00"**  
            **}**  
          **}**  
        **},**  
        **{**  
          **"fulfillment\_state":**  
          **{**  
            **"descriptor":**  
            **{**  
              **"code":"Out-for-delivery",**  
              **"short\_desc":"011,012,013,014,015"**  
            **}**  
          **},**  
          **"cancellation\_fee":**  
          **{**  
            **"percentage":"100.00",**  
            **"amount":**  
            **{**  
              **"currency":"INR",**  
              **"value":"50.00"**  
            **}**  
          **}**  
        **}**  
      **\],**  
      **"tags":**  
      **\[**  
        **{**  
          **"code":"bpp\_terms",**  
          **"list":**  
          **\[**  
            **{**  
              **"code":"max\_liability",**  
              **"value":"2"**  
            **},**  
            **{**  
              **"code":"max\_liability\_cap",**  
              **"value":"10000"**  
            **},**  
            **{**  
              **"code":"mandatory\_arbitration",**  
              **"value":"false"**  
            **},**  
            **{**  
              **"code":"court\_jurisdiction",**  
              **"value":"Bengaluru"**  
            **},**  
            **{**  
              **"code":"delay\_interest",**  
              **"value":"1000"**  
            **},**  
            **{**  
              **"code":"static\_terms",**  
              **"value":"https://github.com/ONDC-Official/protocol-network-extension/discussions/79"**  
            **}**  
          **\]**  
        **},**  
        **{**  
          **"code":"bap\_terms",**  
          **"list":**  
          **\[**  
            **{**  
              **"code":"accept\_bpp\_terms",**  
              **"value":"Y"**  
            **}**  
          **\]**  
        **}**  
      **\],**  
      **"created\_at[^126]":"2023-02-06T22:00:00.000Z",**  
      **"updated\_at":"2023-02-06T22:00:30.000Z"**  
    }  
  }  
}

**N.B.**

1. /on\_confirm returns the fulfillment slot, agent details, vehicle details, E-way bill no, as applicable, for inter-state shipments;

#### 

#### Rules for order confirmation {#rules-for-order-confirmation}

1. /confirm and /on\_confirm calls should be idempotent;

2. Logistics Buyer NP (LBNP) will create the order id which will be unique across the network. In response to /confirm from LBNP, LSP sends /on\_confirm with the same order\_id;

3. Logistics Buyer NP (LBNP) sends /confirm, LSP validates[^127] order:  
   1. if validation successful, LSP responds with ACK and creates order;  
   2. if validation not successful, LSP responds with NACK with error code **66002** (**order validation failure**):  
      1. after LBNP receives NACK, they should cancel the order with [reason code](https://docs.google.com/spreadsheets/d/1_qAtG6Bu2we3AP6OpXr4GVP3X-32v2xNRNSYQhhR6kA/edit#gid=1095388031) **996 (order cancelled because of order confirmation failure)**;  
      2. if this order is passed in subsequent API calls (/confirm, /status, /cancel, /update, etc.), LSP returns NACK with error code 66004 (order not found);  
   3. LSP may also respond with NACK due to other internal errors (http 504, retriable error code **66001**):  
      1. LBNP should retry[^128] /confirm. If after the retry interval, LBNP doesn’t receive ACK/NACK, they should cancel the order with reason code **996**;  
   4. if LBNP doesn’t receive ACK/NACK, they should retry /confirm (same as above). If within the retry interval, LBNP doesn’t receive ACK/NACK, they should cancel the order with reason code **996**;  
4. LSP sends /on\_confirm, LBNP validates[^129] order:  
   1. if validation successful, LBNP responds with ACK;  
   2. if validation not successful, LBNP responds with NACK with error code **63002 (order validation failure)**:  
      1. after LSP receives NACK, they should cancel the order with reason code **997 (order cancelled because of order confirmation failure)** & push the status change to LBNP;  
      2. if this order (with cancellation state & reason code) is passed in subsequent API calls (/on\_confirm, /on\_status, /on\_cancel, /on\_update), LBNP responds with ACK;  
   3. LBNP may also NACK due to internal errors (http 504/503, retriable error code **63001**):  
      1. LSP should retry[^130] /on\_confirm. If after the retry interval, LSP doesn’t receive ACK/NACK, they should cancel the order with reason code **997**;  
      2. If order is passed in subsequent API calls within the retry window (/status, /track, /cancel, /update), LSP responds with current order state along with error code 66003 (order processing in progress);  
      3. If order is passed in subsequent API calls after cancellation (/status, /cancel, /update), LSP sends response with order state as cancelled;

### /update

{  
  "context":  
  {  
    "domain":"nic2004:60232",  
    "country":"IND",  
    "city":"std:080",  
    "action":"update",  
    "core\_version":"**1.2.0**",  
    "bap\_id":"logistics\_buyer.com",  
    "bap\_uri":"https://logistics\_buyer.com/ondc",  
    "bpp\_id":"lsp.com",  
    "bpp\_uri": "https://lsp.com/ondc",  
    "transaction\_id":"T1",  
    "message\_id":"M4",  
    "timestamp":"2023-06-06T22:30:00.000Z",  
    "ttl":"PT30S"  
  },  
  "message":  
  {  
    "update\_target":"fulfillment",  
    "order":  
    {  
      "id":"O2",  
      **"items":**  
      **\[**  
        **{**  
          **"id":"I1",**  
          **"category\_id":"Same Day Delivery",**  
          **"descriptor":**  
          **{**  
             **"code":"P2P"**  
          **}**  
        **}**  
      **\],**  
      "fulfillments":  
      \[  
        {  
          "id":"1",  
          "type":"**Delivery**",  
          **"@ondc/org/awb\_no[^131]":"1227262193237777",**  
          "start":  
          {  
            "instructions[^132]":  
            {  
              **"code":"2[^133]",**  
              "short\_desc":"value of PCC[^134]"  
              "long\_desc":"additional instructions for pickup",  
              **"additional\_desc[^135]":**  
              **{**  
                **"content\_type":"text/html",**  
                **"url":"https://reverse\_qc\_sop\_form.htm"**  
              **}**  
            },  
            **"authorization[^136]":**  
            **{**  
              **"type":"OTP[^137]",**  
              **"token":"OTP code",**  
              **"valid\_from":"2023-06-06T12:00:00.000Z",**  
              **"valid\_to":"2023-06-06T14:00:00.000Z"**  
            **}**  
          },  
          "end":  
          {  
            "instructions[^138]":  
            {  
              **"code":"2[^139]",**  
              "short\_desc":"value of DCC[^140]",  
              "long\_desc":"additional instructions for delivery"  
            }  
          },  
          **"tags":**  
          **\[**  
            **{**  
              **"code":"state",**  
              **"list":**  
              **\[**  
                **{**  
                  **"code":"ready\_to\_ship",**  
                  **"value":"yes"**  
                **}**  
              **\]**  
            **}**  
          **\]**  
        }  
      \],  
      "@ondc/org/linked\_order[^141]":  
      {  
        "items":  
        \[  
          {  
            "category\_id":"Grocery",  
            "descriptor":  
            {  
              "name":"Atta"  
            },  
            "quantity":  
            {  
              "count":2,  
              "measure":  
              {  
                "unit":"**kilogram**",  
                "value":0.5  
              }  
            },  
            "price":  
            {  
              "currency":"INR",  
              "value":"150.00"  
            }  
          }  
        \],  
        "order":  
        {  
          "id":"O1",  
          "weight[^142]":  
          {  
            "unit":"**kilogram**",  
            "value":1  
          },  
          **"dimensions[^143]":**  
          **{**  
            **"length":**  
            **{**  
              **"unit":"centimeter",**  
              **"value":1**  
            **},**  
            **"breadth":**  
            **{**  
              **"unit":"centimeter",**  
              **"value":1**  
            **},**  
            **"height":**  
            **{**  
              **"unit":"centimeter",**  
              **"value":1**  
            **}**  
          **}**  
        }  
      },  
      **"updated\_at":"2023-06-06T23:00:00.000Z"**  
    }  
  }  
}

**N.B.**

1. /update can be used for any of the following:  
   1. Notify LSP that retail order is ready to ship;  
   2. Provide updated delivery instructions to LSP;  
   3. Update linked order details in case of part return / cancel for retail order;  
   4. **Update authorization details for pickup / delivery**;

### /on\_update

* Following enhancements are included:

  * **Handling weight differential**

    * If LSP determines that there is a weight difference, they can update the differential cost and the weight used in calculation of this cost in /on\_update;  
    * In case of RTO, the differential cost for RTO can be updated in /on\_cancel when the RTO request is initiated;  
    * If logistics buyer NP ACKs /on\_update, it means they accept the differential cost proposed by the LSP. **In this case, the differential cost will be added to forward shipment cost & reflected in cancellation fees, defined in the order cancellation terms, for cases where the order is subsequently cancelled**;  
    * If the logistics buyer NP doesn’t accept the differential cost (in /on\_update or /on\_cancel APIs), they can NACK /on\_update with error code 62504\. In this case, logistics buyer NP will still be responsible for the forward costs as included in /confirm & /on\_confirm and will need to abide by the terms & conditions, for weight differential, as agreed to with the LSP;

{  
  "context":  
  {  
    "domain":"nic2004:60232",  
    "country":"IND",  
    "city":"std:080",  
    "action":"on\_update",  
    "core\_version":"**1.2.0**",  
    "bap\_id":"logistics\_buyer.com",  
    "bap\_uri":"https://logistics\_buyer.com/ondc",  
    "bpp\_id":"lsp.com",  
    "bpp\_uri":"https://lsp.com/ondc",  
    "transaction\_id":"T1",  
    "message\_id":"M4",  
    "timestamp":"2023-06-06T22:30:30.000Z"  
  },  
  "message":  
  {  
    "order":  
    {  
      "id":"O2",  
      "state":"In-progress",  
      **"provider":**  
      **{**  
        **"id":"P1",**  
        **"locations[^144]":**  
        **\[**  
          **{**  
            **"id":"L1"**  
          **}**  
        **\]**  
      **},**  
      "items":  
      \[  
        {  
          "id":"I1",  
          **"fulfillment\_id":"1",**  
          "category\_id":"Same Day Delivery",  
          "descriptor":  
          {  
             "code":"P2P"  
          },  
          **"time":**  
          **{**  
            **"label":"TAT",**  
            **"duration":"PT45M",**  
            **"timestamp":"2023-06-06"**  
          **}**  
        }  
      \],  
      **"quote":**  
      **{**  
        **"price":**  
        **{**  
          **"currency":"INR",**  
          **"value": "88.50"**  
        **},**  
        **"breakup":**  
        **\[**  
          **{**  
            **"@ondc/org/item\_id":"I1",**  
            **"@ondc/org/title\_type":"delivery",**  
            **"price":**  
            **{**  
              **"currency":"INR",**  
              **"value":"50.0"**  
            **}**  
          **},**  
          **{**  
            **"@ondc/org/item\_id":"I1",**  
            **"@ondc/org/title\_type":"tax",**  
            **"price":**  
            **{**  
              **"currency":"INR",**  
              **"value":"9.0"**  
            **}**  
          **},**  
          **{**  
            **"@ondc/org/item\_id":"I1[^145]",**  
            **"@ondc/org/title\_type":"diff",**  
            **"price":**  
            **{**  
              **"currency":"INR",**  
              **"value":"25.0"**  
            **}**  
          **},**  
          **{**  
            **"@ondc/org/item\_id":"I1[^146]",**  
            **"@ondc/org/title\_type":"tax\_diff",**  
            **"price":**  
            **{**  
              **"currency":"INR",**  
              **"value":"4.5"**  
            **}**  
          **}**  
        **\]**  
      **},**  
      "fulfillments":  
      \[  
        {  
          "id":"1",  
          "type":"Delivery",  
          **"state":**  
          **{**  
            **"descriptor":**  
            **{**  
              **"code":"Order-picked-up"**  
            **}**  
          **},**  
          "@ondc/org/awb\_no[^147]":"1227262193237777",  
          "tracking":false,  
          **"start":**  
          **{**  
            **"person":**  
            **{**  
              **"name":"Ramu"**  
            **},**  
            **"location":**  
            **{**  
              **"gps":"12.453544,77.928379",**  
              **"address":**  
              **{**  
                **"name":"name",**  
                **"building":"My house or building name",**  
                **"locality":"My street name",**  
                **"city":"Bengaluru",**  
                **"state":"Karnataka",**  
                **"country":"India",**  
                **"area\_code":"560041"**  
              **}**  
            **},**  
            **"contact":**  
            **{**  
              **"phone":"9886098860",**  
              **"email":"abcd.efgh@gmail.com"**  
            **},**  
            "time":  
            {  
              **"duration":"PT15M",**  
              "range":  
              {  
                "start":"2023-06-06T23:45:00.000Z",  
                "end":"2023-06-07T00:00:00.000Z"  
              },  
              **"timestamp":"2023-06-07T00:00:00.000Z"**  
            },  
            "instructions":  
            {  
              **"code":"2",**  
              **"short\_desc":"value of PCC",**  
              **"long\_desc":"additional instructions for pickup",**  
              **"images":**  
              **\[**  
                **"link to downloadable shipping label (required for P2H2P)",**  
                **"https://lsp.com/pickup\_image.png[^148]",**  
                **"https://lsp.com/rider\_location.png[^149]"**  
              **\],**  
              "additional\_desc[^150]":  
              {  
                "content\_type":"text/html",  
                "url":"https://reverse\_qc\_sop\_form.htm"  
              }  
            }  
          },  
          **"end":**  
          **{**  
            **"person":**  
            **{**  
              **"name":"person\_name"**  
            **},**  
            **"location":**  
            **{**  
              **"gps":"12.453544,77.928379",**  
              **"address":**  
              **{**  
                **"name":"My house or building \#",**  
                **"building":"My house or building name",**  
                **"locality":"My street name",**  
                **"city":"Bengaluru",**  
                **"state":"Karnataka",**  
                **"country":"India",**  
                **"area\_code":"560076"**  
              **}**  
            **},**  
            **"contact[^151]":**  
            **{**  
              **"phone":"9886098860",**  
              **"email":"abcd.efgh@gmail.com"**  
            **},**  
            **"instructions[^152]":**  
            **{**  
              **"code":"3",**  
              **"short\_desc":"value of DCC",**  
              **"long\_desc":"additional instructions for delivery",**  
              **"images":**  
              **\[**  
                **"https://lsp.com/delivery\_image.png",**  
                **"https://lsp.com/rider\_location.png"**  
              **\]**  
            **},**  
            **"time":**  
            **{**  
              **"range":**  
              **{**  
                **"start":"2023-06-07T02:00:00.000Z",**  
                **"end":"2023-06-07T02:15:00.000Z"**  
              **}**  
            **}**  
          **},**  
          "agent[^153]":  
          {  
            "name":"person\_name",  
            "phone":"9886098860"  
          },  
          **"vehicle":**  
          **{**  
            **"registration":"3LVJ945"**  
          **},**  
          "@ondc/org/ewaybillno[^154]":"EBN1",  
          "@ondc/org/ebnexpirydate":"2023-06-30T12:00:00.000Z"  
        }  
      \],  
      **"billing":**  
      **{**  
        **"name":"ONDC sellerNP",**  
        **"address":**  
        **{**  
          **"name":"My building no",**  
          **"building":"My building name",**  
          **"locality":"My street name",**  
          **"city":"Bengaluru",**  
          **"state":"Karnataka",**  
          **"country":"India",**  
          **"area\_code":"560076"**  
        **},**  
        **"tax\_number":"XXXXXXXXXXXXXXX",**  
        **"phone":"9886098860",**  
        **"email":"abcd.efgh@gmail.com",**  
        **"created\_at":"2023-06-06T21:30:00.000Z",**  
        **"updated\_at":"2023-06-06T21:30:00.000Z"**  
      **},**  
      **"payment":**  
      **{**  
        **"@ondc/org/collection\_amount":"300.00",**  
        **"collected\_by":"BPP",**  
        **"type":"ON-FULFILLMENT",**  
        **"@ondc/org/settlement\_details":**  
        **\[**  
          **{**  
            **"settlement\_counterparty":"buyer-app",**  
            **"settlement\_type":"upi",**  
            **"upi\_address":"gft@oksbi",**  
            **"settlement\_bank\_account\_no":"XXXXXXXXXX",**  
            **"settlement\_ifsc\_code":"XXXXXXXXX"**  
          **}**  
        **\]**  
      **},**  
      **"@ondc/org/linked\_order":**  
      **{**  
        **"items":**  
        **\[**  
          **{**  
            **"category\_id":"Grocery",**  
            **"descriptor":**  
            **{**  
              **"name":"Atta"**  
            **},**  
            **"quantity":**  
            **{**  
              **"count":2,**  
              **"measure":**  
              **{**  
                **"unit":"kilogram",**  
                **"value":0.5**  
              **}**  
            **},**  
            **"price":**  
            **{**  
              **"currency":"INR",**  
              **"value":"150.00"**  
            **}**  
          **}**  
        **\],**  
        **"provider":**  
        **{**  
          **"descriptor":**  
          **{**  
            **"name":"Aadishwar Store"**  
          **},**  
          **"address":**  
          **{**  
            **"name":"KHB Towers",**  
            **"building":"Building or House No",**  
            **"street":"6th Block",**  
            **"locality":"Koramangala",**  
            **"city":"Bengaluru",**  
            **"state":"Karnataka",**  
            **"area\_code":"560070"**  
          **}**  
        **},**  
        **"order":**  
        **{**  
          **"id":"O1",**  
          **"weight":**  
          **{**  
            **"unit":"kilogram",**  
            **"value":1**  
          **},**  
          **"dimensions":**  
          **{**  
            **"length":**  
            **{**  
              **"unit":"centimeter",**  
              **"value":1**  
            **},**  
            **"breadth":**  
            **{**  
              **"unit":"centimeter",**  
              **"value":1**  
            **},**  
            **"height":**  
            **{**  
              **"unit":"centimeter",**  
              **"value":1**  
            **}**  
          **}**  
        **}**  
      **},**  
      **"tags":**  
      **\[**  
        **{**  
          **"code":"diff\_dim[^155]",**  
          **"list":**  
          **\[**  
            **{**  
              **"code":"unit",**  
              **"value":"centimeter"**  
            **},**  
            **{**  
              **"code":"length",**  
              **"value":"1.5"**  
            **},**  
            **{**  
              **"code":"breadth",**  
              **"value":"1.5"**  
            **},**  
            **{**  
              **"code":"height",**  
              **"value":"1.5"**  
            **}**  
          **\]**  
        **},**  
        **{**  
          **"code":"diff\_weight[^156]",**  
          **"list":**  
          **\[**  
            **{**  
              **"code":"unit",**  
              **"value":"kilogram"**  
            **},**  
            **{**  
              **"code":"weight",**  
              **"value":"1.5"**  
            **}**  
          **\]**  
        **},**  
        **{**  
          **"code":"diff\_proof[^157]",**  
          **"list":**  
          **\[**  
            **{**  
              **"code":"type",**  
              **"value":"image"**  
            **},**  
            **{**  
              **"code":"url",**  
              **"value":"https://lsp.com/sorter/images1.png"**  
            **}**  
          **\]**  
        **}**  
      **\],**  
      "updated\_at":"2023-06-07T23:00:30.000Z"  
    }  
  }  
}

**N.B.**

1. Differential weight update:  
   1. If Logistics buyer NP ACKs /on\_update, it means they accept the updated quote, including the weight differential cost;  
   2. If Logistics buyer NP doesn’t accept the updated quote, including the weight differential cost, they can NACK /on\_update (with error code 62504). In this case, the LSP should revert the proposed changes in quote, order weight & dimensions and subsequent action will be as per the terms & conditions agreed upon between the logistics buyer NP & LSP;

### /cancel

{  
  "context":  
  {  
    "domain":"nic2004:60232",  
    "country":"IND",  
    "city":"std:080",  
    "action":"cancel",  
    "core\_version":"**1.2.0**",  
    "bap\_id":"logistics\_buyer.com",  
    "bap\_uri":"https://logistics\_buyer.com/ondc",  
    "bpp\_id":"lsp.com",  
    "bpp\_uri": "https://lsp.com/ondc",  
    "transaction\_id":"T1",  
    "message\_id":"M5",  
    "timestamp":"2023-06-06T23:00:00.000Z",  
    "ttl":"PT30S"  
  },  
  "message":  
  {  
    "order\_id":"O2",  
    "cancellation\_reason\_id":"011[^158]"  
  }  
}

**N.B.**

1. /cancel will be used by the logistics buyer to cancel the order. LSP can cancel the order through unsolicited /on\_cancel;

2. If cancellation reason is invalid, LSP can NACK /cancel with error code 60009;

3. For cancellation due to TAT breach (cancellation\_reason\_id \= "007"), if LSP determines that fulfillment TAT is not breached, they can NACK /cancel with error code 60010;

   ### /on\_cancel

{  
  "context":  
  {  
    "domain":"nic2004:60232",  
    "country":"IND",  
    "city":"std:080",  
    "action":"on\_cancel",  
    "core\_version":"**1.2.0**",  
    "bap\_id":"logistics\_buyer.com",  
    "bap\_uri":"https://logistics\_buyer.com/ondc",  
    "bpp\_id":"lsp.com",  
    "bpp\_uri":"https://lsp.com/ondc",  
    "transaction\_id":"T1",  
    "message\_id":"M5",  
    "timestamp":"2023-06-06T23:00:30.000Z"  
  },  
  "message":  
  {  
    "order":  
    {  
      "id":"O2",  
      "state":"Cancelled",  
      **"cancellation":**  
      **{**  
        **"cancelled\_by":"logistics\_buyer.com",**  
        **"reason":**  
        **{**  
          **"id":"011"**  
        **}**  
      **},**  
      **"provider":**  
      **{**  
        **"id":"P1",**  
        **"locations":**  
        **\[**  
          **{**  
            **"id":"L1"**  
          **}**  
        **\]**  
      **},**  
      **"items":**  
      **\[**  
        **{**  
          **"id":"I1",**  
          **"fulfillment\_id":"1",**  
          **"category\_id":"Same Day Delivery",**  
          **"descriptor":**  
          **{**  
             **"code":"P2P"**  
          **},**  
          **"time":**  
          **{**  
            **"label":"TAT",**  
            **"duration":"PT45M",**  
            **"timestamp":"2023-06-06"**  
          **}**  
        **}**  
      **\],**  
      **"quote[^159]":**  
      **{**  
        **"price":**  
        **{**  
          **"currency":"INR",**  
          **"value":".."**  
        **},**  
        **"breakup":**  
        **\[**  
          **{**  
            **"@ondc/org/item\_id":"I1",**  
            **"@ondc/org/title\_type":"delivery",**  
            **"price":**  
            **{**  
              **"currency":"INR",**  
              **"value":".."**  
            **}**  
          **},**  
          **{**  
            **"@ondc/org/item\_id":"I1",**  
            **"@ondc/org/title\_type":"tax",**  
            **"price":**  
            **{**  
              **"currency":"INR",**  
              **"value":".."**  
            **}**  
          **}**  
        **\]**  
      **},**  
      "fulfillments":  
      \[  
        {  
          "id":"1",  
          "type":"**Delivery**",  
          "state":  
          {  
            "descriptor":  
            {  
              "code":"Cancelled"  
            }  
          },  
          **"@ondc/org/awb\_no":"1227262193237777[^160]",**  
          **"tracking":false,**  
          **"start":**  
          **{**  
            **"person":**  
            **{**  
              **"name":"person\_name"**  
            **},**  
            **"location":**  
            **{**  
              **"gps":"12.453544,77.928379",**  
              **"address":**  
              **{**  
                **"name":"name",**  
                **"building":"My house or building name",**  
                **"locality":"My street name",**  
                **"city":"Bengaluru",**  
                **"state":"Karnataka",**  
                **"country":"India",**  
                **"area\_code":"560041"**  
              **}**  
            **},**  
            **"contact":**  
            **{**  
              **"phone":"9886098860",**  
              **"email":"abcd.efgh@gmail.com"**  
            **},**  
            **"instructions":**  
            **{**  
              **"code":"2",**  
              **"short\_desc":"value of PCC",**  
              **"long\_desc":"QR code will be attached to package",**  
              **"additional\_desc[^161]":**  
              **{**  
                **"content\_type":"text/html",**  
                **"url":"https://reverse\_qc\_sop\_form.htm"**  
              **}**  
            **},**  
            **"time":**  
            **{**  
              **"duration":"PT15M",**  
              **"range":**  
              **{**  
                **"start":"2023-06-06T22:30:00.000Z",**  
                **"end":"2023-06-06T22:45:00.000Z"**  
              **}**  
            **}**  
          **},**  
          **"end":**  
          **{**  
            **"person":**  
            **{**  
              **"name":"person\_name"**  
            **},**  
            **"location":**  
            **{**  
              **"gps":"12.453544,77.928379",**  
              **"address":**  
              **{**  
                **"name":"My house or building \#",**  
                **"building":"My house or building name",**  
                **"locality":"My street name",**  
                **"city":"Bengaluru",**  
                **"state":"Karnataka",**  
                **"country":"India",**  
                **"area\_code":"560076"**  
              **}**  
            **},**  
            **"contact":**  
            **{**  
              **"phone":"9886098860",**  
              **"email":"abcd.efgh@gmail.com"**  
            **},**  
            **"instructions":**  
            **{**  
              **"code":"3",**  
              **"short\_desc":"value of DCC"**  
            **},**  
            **"time":**  
            **{**  
              **"range":**  
              **{**  
                **"start":"2023-06-06T23:00:00.000Z",**  
                **"end":"2023-06-06T23:15:00.000Z"**  
              **}**  
            **}**  
          **},**  
          **"agent":**  
          **{**  
            **"name":"agent\_name",**  
            **"phone":"9886098860"**  
          **},**  
          **"vehicle":**  
          **{**  
            **"registration":"3LVJ945"**  
          **},**  
          **"tags":**  
          **\[**  
            **{**  
              **"code":"igm\_request[^162]",**  
              **"list":**  
              **\[**  
                **{**  
                  **"code":"id",**  
                  **"value":"Issue1"**  
                **}**  
              **\]**  
            **},**  
            **{**  
              **"code":"precancel\_state[^163]",**  
              **"list":**  
              **\[**  
                **{**  
                  **"code":"fulfillment\_state",**  
                  **"value":"Order-picked-up"**  
                **},**  
                **{**  
                  **"code":"updated\_at",**  
                  **"value":"2023-06-06T23:15:00.000Z"**  
                **}**  
              **\]**  
            **}**  
          **\]**  
        **}**  
      **\],**  
      **"billing":**  
      **{**  
        **"name":"ONDC Logistics Buyer NP",**  
        **"address":**  
        **{**  
          **"name":"My building no",**  
          **"building":"My building name",**  
          **"locality":"My street name",**  
          **"city":"Bengaluru",**  
          **"state":"Karnataka",**  
          **"country":"India",**  
          **"area\_code":"560076"**  
        **},**  
        **"tax\_number":"XXXXXXXXXXXXXXX",**  
        **"phone":"9886098860",**  
        **"email":"abcd.efgh@gmail.com",**  
        **"created\_at":"2023-06-06T21:30:00.000Z",**  
        **"updated\_at":"2023-06-06T21:30:00.000Z"**  
      **},**  
      **"payment":**  
      **{**  
        **"@ondc/org/collection\_amount":"300.00",**  
        **"collected\_by":"BPP",**  
        **"type":"ON-FULFILLMENT",**  
        **"@ondc/org/settlement\_details":**  
        **\[**  
          **{**  
            **"settlement\_counterparty":"buyer-app",**  
            **"settlement\_type":"upi",**  
            **"upi\_address":"gft@oksbi",**  
            **"settlement\_bank\_account\_no":"XXXXXXXXXX",**  
            **"settlement\_ifsc\_code":"XXXXXXXXX"**  
          **}**  
        **\]**  
      **},**  
      **"@ondc/org/linked\_order":**  
      **{**  
        **"items":**  
        **\[**  
          **{**  
            **"category\_id":"Grocery",**  
            **"descriptor":**  
            **{**  
              **"name":"Atta"**  
            **},**  
            **"quantity":**  
            **{**  
              **"count":2,**  
              **"measure":**  
              **{**  
                **"unit":"kilogram",**  
                **"value":0.5**  
              **}**  
            **},**  
            **"price":**  
            **{**  
              **"currency":"INR",**  
              **"value":"150.00"**  
            **}**  
          **}**  
        **\],**  
        **"provider":**  
        **{**  
          **"descriptor":**  
          **{**  
            **"name":"Aadishwar Store"**  
          **},**  
          **"address":**  
          **{**  
            **"name":"KHB Towers",**  
            **"building":"Building or House No",**  
            **"locality":"Koramangala",**  
            **"city":"Bengaluru",**  
            **"state":"Karnataka",**  
            **"area\_code":"560070"**  
          **}**  
        **},**  
        **"order":**  
        **{**  
          **"id":"O1",**  
          **"weight":**  
          **{**  
            **"unit":"kilogram",**  
            **"value":1**  
          **},**  
          **"dimensions":**  
          **{**  
            **"length":**  
            **{**  
              **"unit":"centimeter",**  
              **"value":1**  
            **},**  
            **"breadth":**  
            **{**  
              **"unit":"centimeter",**  
              **"value":1**  
            **},**  
            **"height":**  
            **{**  
              **"unit":"centimeter",**  
              **"value":1**  
            **}**  
          **}**  
        **}**  
      **},**  
      **"updated\_at":"2023-06-06T22:00:30.000Z"**  
    }  
  }  
}

###  /on\_cancel (RTO flow)

RTO will be initiated when the end buyer does not accept or cannot be located for the order / fulfillment to be delivered;

In case of RTO:

* Every fulfillment will have an identifier for whether RTO results in actual return to origin (in /confirm);

  * RTO, with return, will result in package being returned to the point of pickup for now (will be amended to allow merchant to provide return location which may be different from point of pickup);

  * Buyers are within their rights to RTO, if the fulfillment is delivered beyond the fulfillment TAT, thus constituting a TAT breach;

  * In the event of RTO, without any breach of TAT, the Logistics Buyer NP may recover both the forward shipping cost as well as RTO costs from the retail buyer NP. In this case, the LSP may be expected to help establish that the retail buyer refused to accept or was not available / couldn’t be contacted to accept delivery;

  RTO process flow is as follows:

* LSP tries to deliver the order to the retail buyer (it is assumed that a certain number of retries are built in for all deliveries except immediate delivery). Typical number of retries is 3 as per industry standard;

* Retail buyer does not accept the order or is unreachable / unavailable at the mentioned shipping address on attempt to deliver the order (LSP may be expected to help substantiate that reasonable attempt was made to connect and deliver the order to the retail buyer at the specified address);

* LSP initiates cancellation of logistics order, through /on\_cancel, and provides specific reason code as defined, for [hyperlocal](https://docs.google.com/spreadsheets/d/1_qAtG6Bu2we3AP6OpXr4GVP3X-32v2xNRNSYQhhR6kA/edit?gid=610954815#gid=610954815) & [inter-city](https://docs.google.com/spreadsheets/d/1_qAtG6Bu2we3AP6OpXr4GVP3X-32v2xNRNSYQhhR6kA/edit?gid=956300231#gid=956300231), and AWB no (for forward shipment, if applicable);

* /on\_cancel with reason code (that triggers RTO as defined in the reason codes) will initiate RTO and result in the following:

  * addition of new fulfillment record for the logistics order, with fulfillment start location / time, fulfillment state set to "RTO-Initiated";

  * quote for RTO will be added to the original quote (for forward shipment), matching the quote provided in logistics /on\_search;

  * order state will be set to "Cancelled";

* If the RTO request is valid (as determined by the reason code and above info provided), the logistics buyer NP has to accept the request. If LSP sends a different quote, from what was sent in /on\_search, the logistics buyer NP may reject the RTO request (with NACK and error code 62503\) and the RTO request has to be resent with the correct quote;

* /on\_cancel will also result in cascaded retail /on\_cancel request, for the fulfillment, and include the new fulfillment record for RTO, fulfillment start location / time, fulfillment state, order state and quote for RTO (if retail buyer has to pay for the RTO cost as per the cancellation reason code);

* After RTO delivery is completed, LSP updates fulfillment status for RTO fulfillment to "RTO-Delivered", fulfillment end time, etc. through /on\_status and this is cascaded to the retail order;

* In case of RTO without return, fulfillment state will be updated to "RTO-Disposed";

{  
  "context":  
  {  
    "domain":"nic2004:60232",  
    "country":"IND",  
    "city":"std:080",  
    "action":"on\_cancel",  
    "core\_version":"**1.2.0**",  
    "bap\_id":"logistics\_buyer.com",  
    "bap\_uri":"https://logistics\_buyer.com/ondc",  
    "bpp\_id":"lsp.com",  
    "bpp\_uri":"https://lsp.com/ondc",  
    "transaction\_id":"T1",  
    "message\_id":"M5",  
    "timestamp":"2023-06-06T23:00:30.000Z"  
  },  
  "message":  
  {  
    "order":  
    {  
      "id":"O2",  
      "state":"Cancelled",  
      "cancellation":  
      {  
        "cancelled\_by":"lsp.com",  
        "reason":  
        {  
          "id":"013"  
        }  
      },  
      "provider":  
      {  
        "id":"P1",  
        "locations":  
        \[  
          {  
            "id":"L1"  
          }  
        \]  
      },  
      "items":  
      \[  
        {  
          "id":"I1",  
          "fulfillment\_id":"1",  
          "category\_id":"Same Day Delivery",  
          "descriptor":  
          {  
             "code":"P2P"  
          },  
          "time":  
          {  
            "label":"TAT",  
            "duration":"PT45M",  
            "timestamp":"2023-06-06"  
          }  
        },  
        {  
          "id":"I2",  
          "fulfillment\_id":"1-RTO",  
          "category\_id":"Same Day Delivery",  
          "descriptor":  
          {  
             "code":"P2P"  
          },  
          "time":  
          {  
            "label":"TAT",  
            "duration":"PT45M",  
            "timestamp":"2023-06-06"  
          }  
        }  
      \],  
      "quote[^164]":  
      {  
        "price":  
        {  
          "currency":"INR",  
          "value":"82.60"  
        },  
        "breakup":  
        \[  
          {  
            "@ondc/org/item\_id":"I1",  
            "@ondc/org/title\_type":"delivery",  
            "price":  
            {  
              "currency":"INR",  
              "value":"50.00"  
            }  
          },  
          {  
            "@ondc/org/item\_id":"I1",  
            "@ondc/org/title\_type":"tax",  
            "price":  
            {  
              "currency":"INR",  
              "value":"9.00"  
            }  
          },  
          **{**  
            "@ondc/org/item\_id":"I2",  
            **"@ondc/org/title\_type":"rto",**  
            **"price":**  
            **{**  
              **"currency":"INR",**  
              **"value":"20.0[^165]"**  
            **}**  
          **},**  
          **{**  
            **"@ondc/org/item\_id":"I2",**  
            **"@ondc/org/title\_type":"tax",**  
            **"price":**  
            **{**  
              **"currency":"INR",**  
              **"value":"3.60[^166]"**  
            **}**  
          **}**  
        \]  
      },  
      "fulfillments":  
      \[  
        {  
          "id":"1",  
          "type":"Delivery",  
          "state":  
          {  
            "descriptor":  
            {  
              "code":"Pending"  
            }  
          },  
          "@ondc/org/awb\_no":"1227262193237777",  
          "tracking":false,  
          "start":  
          {  
            "person":  
            {  
              "name":"person\_name"  
            },  
            "location":  
            {  
              "gps":"12.453544,77.928379",  
              "address":  
              {  
                "name":"name",  
                "building":"My house or building name",  
                "locality":"My street name",  
                "city":"Bengaluru",  
                "state":"Karnataka",  
                "country":"India",  
                "area\_code":"560041"  
              }  
            },  
            "contact":  
            {  
              "phone":"9886098860",  
              "email":"abcd.efgh@gmail.com"  
            },  
            "instructions":  
            {  
              "code":"2",  
              "short\_desc":"value of PCC",  
              "long\_desc":"QR code will be attached to package",  
              "additional\_desc":  
              {  
                "content\_type":"text/html",  
                "url":"https://reverse\_qc\_sop\_form.htm"  
              }  
            },  
            "time":  
            {  
              "duration":"PT15M",  
              "range":  
              {  
                "start":"2023-06-06T22:30:00.000Z",  
                "end":"2023-06-06T22:45:00.000Z"  
              }  
            }  
          },  
          "end":  
          {  
            "person":  
            {  
              "name":"person\_name"  
            },  
            "location":  
            {  
              "gps":"12.4535445,77.9283792",  
              "address":  
              {  
                "name":"My house or building \#",  
                "building":"My house or building name",  
                "locality":"My street name",  
                "city":"Bengaluru",  
                "state":"Karnataka",  
                "country":"India",  
                "area\_code":"560076"  
              }  
            },  
            "contact":  
            {  
              "phone":"9886098860",  
              "email":"abcd.efgh@gmail.com"  
            },  
            "instructions":  
            {  
              "code":"3",  
              "short\_desc":"value of DCC"  
            },  
            "time":  
            {  
              "range":  
              {  
                "start":"2023-06-06T23:00:00.000Z",  
                "end":"2023-06-06T23:15:00.000Z"  
              }  
            }  
          },  
          "agent":  
          {  
            "name":"agent\_name",  
            "phone":"9886098860"  
          },  
          "vehicle":  
          {  
            "registration":"3LVJ945"  
          },  
          "@ondc/org/ewaybillno":"EBN1",  
          "@ondc/org/ebnexpirydate":"2023-06-30T12:00:00.000Z",  
          **"tags":**  
          **\[**  
            **{**  
              **"code":"rto\_event",**  
              **"list":**  
              **\[**  
                **{**  
                  **"code":"retry\_count",**  
                  **"value":"3[^167]"**  
                **},**  
                **{**  
                  **"code":"rto\_id",**  
                  **"value":"F1-RTO[^168]"**  
                **},**  
                **{**  
                  **"code":"cancellation\_reason\_id",**  
                  **"value":"013[^169]"**  
                **},**  
                **{**  
                  **"code":"sub\_reason\_id",**  
                  **"value":"004[^170]"**  
                **},**  
                **{**  
                  **"code":"cancelled\_by",**  
                  **"value":"lsp.com[^171]"**  
                **}**  
              **\]**  
            **},**  
            **{**  
              **"code":"igm\_request[^172]",**  
              **"list":**  
              **\[**  
                **{**  
                  **"code":"id",**  
                  **"value":"Issue1"**  
                **}**  
              **\]**  
            **},**  
            **{**  
              **"code":"precancel\_state[^173]",**  
              **"list":**  
              **\[**  
                **{**  
                  **"code":"fulfillment\_state",**  
                  **"value":"Order-picked-up"**  
                **},**  
                **{**  
                  **"code":"updated\_at",**  
                  **"value":"2023-06-06T23:15:00.000Z"**  
                **}**  
              **\]**  
            **}**  
          **\]**  
        },  
        **{**  
          **"id":"1-RTO",**  
          **"type":"RTO",**  
          **"state":**  
          **{**  
            **"descriptor":**  
            **{**  
              **"code":"RTO-Initiated[^174]"**  
            **}**  
          **},**  
          **"start":**  
          **{**  
            **"time":**  
            **{**  
              **"timestamp":"2023-06-07T14:00:00.000Z"**  
            **}**  
          **}**  
        }  
      \],  
      "billing":  
      {  
        "name":"ONDC sellerNP",  
        "address":  
        {  
          "name":"My building no",  
          "building":"My building name",  
          "locality":"My street name",  
          "city":"Bengaluru",  
          "state":"Karnataka",  
          "country":"India",  
          "area\_code":"560076"  
        },  
        "tax\_number":"XXXXXXXXXXXXXXX",  
        "phone":"9886098860",  
        "email":"abcd.efgh@gmail.com",  
        "created\_at":"2023-06-06T21:30:00.000Z",  
        "updated\_at":"2023-06-06T21:30:00.000Z"  
      },  
      "payment":  
      {  
        "@ondc/org/collection\_amount":"300.00",  
        "collected\_by":"BPP",  
        "type":"ON-FULFILLMENT",  
        "@ondc/org/settlement\_details":  
        \[  
          {  
            "settlement\_counterparty":"buyer-app",  
            "settlement\_type":"upi",  
            "upi\_address":"gft@oksbi",  
            "settlement\_bank\_account\_no":"XXXXXXXXXX",  
            "settlement\_ifsc\_code":"XXXXXXXXX"  
          }  
        \]  
      },  
      "@ondc/org/linked\_order":  
      {  
        "items":  
        \[  
          {  
            "category\_id":"Grocery",  
            "descriptor":  
            {  
              "name":"Atta"  
            },  
            "quantity":  
            {  
              "count":2,  
              "measure":  
              {  
                "unit":"kilogram",  
                "value":0.5  
              }  
            },  
            "price":  
            {  
              "currency":"INR",  
              "value":"150.00"  
            }  
          }  
        \],  
        "provider":  
        {  
          "descriptor":  
          {  
            "name":"Aadishwar Store"  
          },  
          "address":  
          {  
            "name":"KHB Towers",  
            "building":"Building or House No",  
            "locality":"Koramangala",  
            "city":"Bengaluru",  
            "state":"Karnataka",  
            "area\_code":"560070"  
          }  
        },  
        "order":  
        {  
          "id":"O1",  
          "weight":  
          {  
            "unit":"kilogram",  
            "value":1  
          },  
          "dimensions":  
          {  
            "length":  
            {  
              "unit":"centimeter",  
              "value":1  
            },  
            "breadth":  
            {  
              "unit":"centimeter",  
              "value":1  
            },  
            "height":  
            {  
              "unit":"centimeter",  
              "value":1  
            }  
          }  
        }  
      },  
      "created\_at":"2023-06-06T22:00:00.000Z",  
      "updated\_at":"2023-06-06T22:00:30.000Z"  
    }  
  }  
}

**N.B.**

1. In case of weight differential impacting RTO costs, LSP needs to provide the differential cost & updated weight / dimensions as in /on\_update;  
2. If Logistics buyer NP ACKs /on\_cancel, it means they accept the updated quote, including the weight differential cost;  
3. If Logistics buyer NP doesn’t accept the updated quote, including the weight differential cost, they can NACK /on\_cancel (with error code 62504). In this case, the LSP should revert the proposed changes in differential cost in the quote, updated order weight & dimensions and subsequent action will be as per the terms & conditions agreed upon between the logistics buyer NP & LSP;

   ### 

   ### /track

* Tracking may be enabled by retail buyer NPs for orders;  
* To enable fulfillment tracking:  
  * Tracking should be enabled for the fulfillment;  
  * Fulfillment tracking becomes active after order is picked up for delivery and becomes inactive after the fulfillment is delivered;  
  * /track request & /on\_track response are cascaded from retail seller NP (logistics buyer NP) to LSP & back;

{  
  "context":  
  {  
    "domain":"nic2004:60232",  
    "action":"track",  
    "country":"IND",  
    "city":"std:080",  
    "core\_version":"**1.2.0**",  
    "bap\_id":"logistics\_buyer.com",  
    "bap\_uri":"https://logistics\_buyer.com/ondc",  
    "bpp\_id":"lsp.com",  
    "bpp\_uri":"https://lsp.com/ondc",  
    "transaction\_id":"T1",  
    "message\_id":"M6",  
    "timestamp":"2023-06-06T23:30:00.000Z",  
    "ttl":"PT30S"  
  },  
  "message":  
  {  
    "order\_id":"O2"  
  }  
}

**N.B.**

1. Buyer NP will need to call /track to poll near real-time gps coordinates for the rider;

2. /track should be called only after the rider is assigned for the order. If /track is called for an order for which the order is not yet picked up or order has been picked up but fulfillment tracking is disabled, the LSP must return NACK with error code 60012;

### /on\_track

{  
  "context":  
  {  
    "domain":"nic2004:60232",  
    "action":"on\_track",  
    "country":"IND",  
    "city":"std:080",  
    "core\_version":"**1.2.0**",  
    "bap\_id":"logistics\_buyer.com",  
    "bap\_uri":"https://logistics\_buyer.com/ondc",  
    "bpp\_id":"lsp.com",  
    "bpp\_uri":"https://lsp.com/ondc",  
    "transaction\_id":"T1",  
    "message\_id":"M6",  
    "timestamp":"2023-06-06T23:30:30.000Z",  
    "ttl":"PT30S"  
  },  
  "message":  
  {  
    "tracking":  
    {  
      "id":"F1[^175]",  
      "url":"https://lsp.com/ondc/track/F1[^176]",  
      "location":  
      {  
        "gps[^177]":"12.974002,77.613458",  
        "time":  
        {  
          "timestamp[^178]":"2023-06-06T23:30:00.000Z"  
        },  
        "updated\_at[^179]":"2023-06-06T23:31:00.000Z"  
      },  
      "status[^180]":"active",  
      "tags[^181]":  
      \[  
        **{**  
          **"code":"order[^182]",**  
          **"list":**  
          **\[**  
            **{**  
              **"code":"id",**  
              **"value":"O2"**  
            **}**  
          **\]**  
        **},**  
        **{**  
          **"code":"config[^183]",**  
          **"list":**  
          **\[**  
            **{**  
              **"code":"attr",**  
              **"value":"tracking.location.gps[^184]"**  
            **},**  
            **{**  
              **"code":"type",**  
              **"value":"live\_poll[^185]"**  
            **}**  
          \]  
        },  
        {  
          "code":"path",  
          "list":  
          \[  
            {  
              "code":"lat\_lng",  
              "value":"12.974002,77.613458"  
            },  
            {  
              "code":"sequence",  
              "value":"1"  
            }  
          \]  
        },  
        {  
          "code":"path",  
          "list":  
          \[  
            {  
              "code":"lat\_lng",  
              "value":"12.974077,77.613600"  
            },  
            {  
              "code":"sequence",  
              "value":"2"  
            }  
          \]  
        },  
        {  
          "code":"path",  
          "list":  
          \[  
            {  
              "code":"lat\_lng",  
              "value":"12.974098,77.613699"  
            },  
            {  
              "code":"sequence",  
              "value":"3"  
            }  
          \]  
        }  
      \]  
    }  
  }  
}

**N.B.**

1. To enable tracking:  
* Tracking should be enabled for the fulfillment;  
* Tracking becomes active after rider is assigned for the order and becomes inactive after the fulfillment is delivered;  
* /on\_track will be solicited response as explained above;  
* /track request & /on\_track response are cascaded from retail seller NP to LSP & back;

  ### /status

{  
  "context":  
  {  
    "domain":"nic2004:60232",  
    "country":"IND",  
    "city":"std:080",  
    "action":"status",  
    "core\_version":"**1.2.0**",  
    "bap\_id":"logistics\_buyer.com",  
    "bap\_uri":"https://logistics\_buyer.com/ondc",  
    "bpp\_id":"lsp.com",  
    "bpp\_uri":"https://lsp.com/ondc",  
    "transaction\_id":"T1",  
    "message\_id":"M7",  
    "timestamp":"2023-06-07T00:00:00.000Z",  
    "ttl": "PT30S"  
  },  
  "message":  
  {  
    "order\_id":"O2"  
  }  
}

### /on\_status

* Following enhancements are included:

  * **Proof of pickup / delivery**

    * Pickup & delivery are generally linked with pickup confirmation code (PCC) and delivery confirmation code (DCC);  
    * In some cases, pickup & delivery can be authenticated using mechanisms such as OTP. In other cases, proof of pickup & delivery can optionally be communicated over the protocol;  
    * Proof could include images (for pickup, delivery), last gps coordinates of rider, etc.;

  * Additional fulfillment states for hyperlocal, intercity;

{  
  "context":  
  {  
    "domain":"nic2004:60232",  
    "country":"IND",  
    "city":"std:080",  
    "action":"on\_status",  
    "core\_version":"**1.2.0**",  
    "bap\_id":"logistics\_buyer.com",  
    "bap\_uri":"https://logistics\_buyer.com/ondc",  
    "bpp\_id":"lsp.com",  
    "bpp\_uri": "https://lsp.com/ondc",  
    "transaction\_id":"T1",  
    "message\_id":"M7",  
    "timestamp":"2023-06-07T00:00:30.000Z"  
  },  
  "message":  
  {  
    "order":  
    {  
      "id":"O2",  
      "state":"Completed[^186]",  
      **"cancellation[^187]":**  
      **{**  
        **"cancelled\_by":"buyerNP.com",**  
        **"reason":**  
        **{**  
          **"id":"011"**  
        **}**  
      **},**  
      "provider":  
      {  
        "id":"P1",  
        "locations[^188]":  
        \[  
          {  
            "id":"L1"  
          }  
        \]  
      },  
      "items":  
      \[  
        {  
          "id":"I1",  
          **"fulfillment\_id":"1",**  
          "category\_id":"Same Day Delivery",  
          "descriptor":  
          {  
             "code":"P2P"  
          },  
          **"time":**  
          **{**  
            **"label":"TAT",**  
            **"duration":"PT45M",**  
            **"timestamp":"2023-06-06"**  
          **}**  
        },  
        {[^189]  
          "id":"I2",  
          **"fulfillment\_id":"1-RTO",**  
          "category\_id":"Same Day Delivery",  
          "descriptor":  
          {  
             "code":"P2P"  
          },  
          **"time":**  
          **{**  
            **"label":"TAT",**  
            **"duration":"PT45M",**  
            **"timestamp":"2023-06-06"**  
          **}**  
        }  
      \],  
      "quote":  
      {  
        "price":  
        {  
          "currency":"INR",  
          "value": "108.50"  
        },  
        "breakup":  
        \[  
          {  
            "@ondc/org/item\_id":"I1",  
            "@ondc/org/title\_type":"**delivery**",  
            "price":  
            {  
              "currency":"INR",  
              "value":"50.00"  
            }  
          },  
          {  
            "@ondc/org/item\_id":"I1",  
            "@ondc/org/title\_type":"**tax**",  
            "price":  
            {  
              "currency":"INR",  
              "value":"9.00"  
            }  
          },  
          {[^190]  
            "@ondc/org/item\_id":"I2",  
            "@ondc/org/title\_type":"**rto**",  
            "price":  
            {  
              "currency":"INR",  
              "value":"20.00"  
            }  
          },  
          **{[^191]**  
            **"@ondc/org/item\_id":"I1",**  
            **"@ondc/org/title\_type":"diff",**  
            **"price":**  
            **{**  
              **"currency":"INR",**  
              **"value":"25.0"**  
            **}**  
          **},**  
          **{[^192]**  
            **"@ondc/org/item\_id":"I1",**  
            **"@ondc/org/title\_type":"tax\_diff",**  
            **"price":**  
            **{**  
              **"currency":"INR",**  
              **"value":"4.5"**  
            **}**  
          **}**  
        \]  
      },  
      "fulfillments":  
      \[  
        {  
          **"id":"1",**  
          "type":"**Delivery**",  
          "@ondc/org/awb\_no":"1227262193237777",  
          "state":  
          {  
            "descriptor":  
            {  
              "code":"Order-picked-up[^193]",  
              **"short\_desc":"pickup or delivery failed reason code[^194]",**  
            }  
          },  
          "tracking":false,  
          "start":  
          {  
            **"person":**  
            **{**  
              **"name":"person\_name"**  
            **},**  
            **"location":**  
            **{**  
              **"gps":"12.453544,77.928379",**  
              **"address":**  
              **{**  
                **"name":"name",**  
                **"building":"My house or building name",**  
                **"locality":"My street name",**  
                **"city":"Bengaluru",**  
                **"state":"Karnataka",**  
                **"country":"India",**  
                **"area\_code":"560041"**  
              **}**  
            **},**  
            **"contact":**  
            **{**  
              **"phone":"9886098860",**  
              **"email":"abcd.efgh@gmail.com"**  
            **},**  
            "time":  
            {  
              **"duration":"PT15M",**  
              "range":  
              {  
                "start":"2023-06-06T22:30:00.000Z",  
                "end":"2023-06-06T22:45:00.000Z"  
              },  
              "timestamp[^195]":"2023-06-06T22:30:00.000Z"  
            },  
            "instructions[^196]":  
            {  
              **"code":"2",**  
              "name":"ONDC order",  
              "short\_desc":"value of PCC",  
              "long\_desc":"additional instructions for pickup[^197]",  
              "images[^198]":  
              \[  
                "link to downloadable shipping label",  
                **"https://lsp.com/pickup\_image.png",**  
                **"https://lsp.com/rider\_location.png"**  
              \],  
              **"additional\_desc[^199]":**  
              **{**  
                **"content\_type":"text/html",**  
                **"url":"https://reverse\_qc\_sop\_form.htm"**  
              **}**  
            },  
            **"authorization[^200]":**  
            **{**  
              **"type":"OTP[^201]",**  
              **"token":"OTP code",**  
              **"valid\_from":"2023-06-07T12:00:00.000Z",**  
              **"valid\_to":"2023-06-07T14:00:00.000Z"**  
            **}**  
          },  
          "end":  
          {  
            **"person":**  
            **{**  
              **"name":"person\_name"**  
            **},**  
            **"location":**  
            **{**  
              **"gps":"12.453544,77.928379",**  
              **"address":**  
              **{**  
                **"name":"My house or building \#",**  
                **"building":"My house or building name",**  
                **"locality":"My street name",**  
                **"city":"Bengaluru",**  
                **"state":"Karnataka",**  
                **"country":"India",**  
                **"area\_code":"560076"**  
              **}**  
            **},**  
            **"contact":**  
            **{**  
              **"phone":"9886098860",**  
              **"email":"abcd.efgh@gmail.com"**  
            **},**  
            "time":  
            {  
              "range":  
              {  
                "start":"2023-06-06T23:00:00.000Z",  
                "end":"2023-06-06T23:15:00.000Z"  
              },  
              "timestamp[^202]":"2023-06-06T23:00:00.000Z"  
            },  
            "instructions[^203]":  
            {  
              **"code":"3",**  
              "name":"ONDC order",  
              "short\_desc":"value of DCC",  
              **"long\_desc":"additional instructions for delivery[^204]",**  
              "images[^205]":  
              \[  
                **"https://lsp.com/delivery\_image.png",**  
                **"https://lsp.com/rider\_location.png"**  
              \]  
            },  
            **"authorization[^206]":**  
            **{**  
              **"type":"OTP[^207]",**  
              **"token":"OTP code",**  
              **"valid\_from":"2023-06-07T12:00:00.000Z",**  
              **"valid\_to":"2023-06-07T14:00:00.000Z"**  
            **}**  
          },  
          "agent[^208]":  
          {  
            "name":"agent\_name",  
            "phone":"9886098860"  
          },  
          "vehicle[^209]":  
          {  
            "registration":"3LVJ945"  
          },  
          "@ondc/org/ewaybillno":"EBN1",  
          "@ondc/org/ebnexpirydate":"2023-06-30T12:00:00.000Z",  
          **"tags":**  
          **\[**  
            **{**  
              **"code":"reverseqc\_output[^210]",**  
              **"list":**  
              **\[**  
                **{**  
                  **"code":"P001",**  
                  **"value":"Atta"**  
                **},**  
                **{**  
                  **"code":"P003",**  
                  **"value":"1"**  
                **},**  
                **{**  
                  **"code":"Q001",**  
                  **"value":"Y[^211]"**  
                **}**  
              **\]**  
            **},**  
            **{**  
              **"code":"fulfillment\_delay[^212]",**  
              **"list":**  
              **\[**  
                **{**  
                  **"code":"state",**  
                  **"value":"Order-picked-up[^213]"**  
                **},**  
                **{**  
                  **"code":"reason\_id",**  
                  **"value":"002[^214]"**  
                **},**  
                **{**  
                  **"code":"timestamp",**  
                  **"value":"2023-06-06T22:00:00.000Z"**  
                **}**  
              **\]**  
            **},**  
            **{**  
              **"code":"tracking[^215]",**  
              **"list":**  
              **\[**  
                **{**  
                  **"code":"gps\_enabled[^216]",**  
                  **"value":"yes"**  
                **},**  
                **{**  
                  **"code":"url\_enabled[^217]",**  
                  **"value":"no"**  
                **},**  
                **{**  
                  **"code":"url[^218]",**  
                  **"value":"https://sellerNP.com/ondc/tracking\_url"**  
                **}**  
              **\]**  
            **}**  
          **\]**  
        },  
        {  
          **"id":"1-RTO",**  
          "type": "RTO",  
          "state":  
          {  
            "descriptor":  
            {  
              "code":"RTO-Initiated"  
            }  
          },  
          "start":  
          {  
            "time":  
            {  
              "range":  
              {  
                "start":"2023-06-06T23:00:00.000Z",  
                "end":"2023-06-06T23:00:00.000Z"  
              },  
              "timestamp[^219]":"2023-06-06T23:00:00.000Z"  
            }  
          },  
          "agent[^220]":  
          {  
            "name":"agent\_name",  
            "phone":"9886098860"  
          }  
        }  
      \],  
      "payment":  
      {  
        **"@ondc/org/collection\_amount":"300.00",**  
        "type":"POST-FULFILLMENT",  
        "collected\_by":"BPP",  
        **"status":"PAID",**  
        **"time[^221]":**  
        **{**  
          **"timestamp": "2023-06-07T10:00:00.000Z"**  
        **},**  
        "@ondc/org/settlement\_details":  
        \[  
          {  
            "settlement\_counterparty":"buyer-app",  
            "settlement\_type":"upi",  
            "upi\_address":"gft@oksbi",  
            "settlement\_bank\_account\_no":"XXXXXXXXXX",  
            "settlement\_ifsc\_code":"XXXXXXXXX",  
            "settlement\_status":"PAID",  
            "settlement\_reference":"XXXXXXXXX",  
            "settlement\_timestamp":"2023-02-10T00:00:00.000Z"  
          }  
        \]  
      },  
      "billing":  
      {  
        "name":"ONDC Seller NP",  
        "address":  
        {  
          "name":"My building \#",  
          **"building":"My building name",**  
          "locality":"My street name",  
          "city":"Bengaluru",  
          "state":"Karnataka",  
          "country":"India",  
          "area\_code":"560076"  
        },  
        "tax\_number":"XXXXXXXXXXXXXXX",  
        "phone":"9886098860",  
        "email":"abcd.efgh@gmail.com"  
      },  
      **"@ondc/org/linked\_order":**  
      **{**  
        **"items":**  
        **\[**  
          **{**  
            **"category\_id":"Grocery",**  
            **"descriptor":**  
            **{**  
              **"name":"Atta"**  
            **},**  
            **"quantity":**  
            **{**  
              **"count":2,**  
              **"measure":**  
              **{**  
                **"unit":"kilogram",**  
                **"value":0.5**  
              **}**  
            **},**  
            **"price":**  
            **{**  
              **"currency":"INR",**  
              **"value":"150.00"**  
            **}**  
          **}**  
        **\],**  
        **"provider":**  
        **{**  
          **"descriptor":**  
          **{**  
            **"name":"Aadishwar Store"**  
          **},**  
          **"address":**  
          **{**  
            **"name":"KHB Towers",**  
            **"building":"Building or House No",**  
            **"street":"6th Block",**  
            **"locality":"Koramangala",**  
            **"city":"Bengaluru",**  
            **"state":"Karnataka",**  
            **"area\_code":"560070"**  
          **}**  
        **},**  
        **"order":**  
        **{**  
          **"id":"O1",**  
          **"weight":**  
          **{**  
            **"unit":"kilogram",**  
            **"value":1**  
          **},**  
          **"dimensions":**  
          **{**  
            **"length":**  
            **{**  
              **"unit":"centimeter",**  
              **"value":1**  
            **},**  
            **"breadth":**  
            **{**  
              **"unit":"centimeter",**  
              **"value":1**  
            **},**  
            **"height":**  
            **{**  
              **"unit":"centimeter",**  
              **"value":1**  
            **}**  
          **}**  
        **}**  
      **},**  
      **"tags":**  
      **\[**  
        **{**  
          **"code":"diff\_dim[^222]",**  
          **"list":**  
          **\[**  
            **{**  
              **"code":"unit",**  
              **"value":"centimeter"**  
            **},**  
            **{**  
              **"code":"length",**  
              **"value":"1.5"**  
            **},**  
            **{**  
              **"code":"breadth",**  
              **"value":"1.5"**  
            **},**  
            **{**  
              **"code":"height",**  
              **"value":"1.5"**  
            **}**  
          **\]**  
        **},**  
        **{**  
          **"code":"diff\_weight[^223]",**  
          **"list":**  
          **\[**  
            **{**  
              **"code":"unit",**  
              **"value":"kilogram"**  
            **},**  
            **{**  
              **"code":"weight",**  
              **"value":"1.5"**  
            **}**  
          **\]**  
        **},**  
        **{**  
          **"code":"diff\_proof[^224]",**  
          **"list":**  
          **\[**  
            **{**  
              **"code":"type",**  
              **"value":"image"**  
            **},**  
            **{**  
              **"code":"url",**  
              **"value":"https://lsp.com/sorter/images1.png"**  
            **}**  
          **\]**  
        **}**  
      **\]**  
    }  
  }  
}

**N.B.**

1. /on\_status response can be solicited or unsolicited;  
2. /on\_status returns the audit trail for order items (including status of item returns, cancels processed or in-progress), the current state of order items & fulfillments, order quote & breakup;  
3. Buyer app can send NACK for /on\_status with error code for cases such as:  
   1. NACK received by seller app for /on\_confirm but seller app sends /on\_status to update order state to "Accepted";  
4. In case of NACK received for /on\_status, seller app should rollback order to previous state of the order, i.e. the state prior to sending /on\_status;

### Fulfillment states & mapping to order states {#fulfillment-states-&-mapping-to-order-states}

1. **Hyperlocal (P2P)**

| Fulfillment State | When to assign state | Order state |
| :---- | :---- | :---- |
| "Pending" | default fulfillment state | "Created" or "Accepted" |
| "Searching-for-Agent" | after RTS | "In-progress" |
| "Agent-assigned" | rider assigned | "In-progress" |
| "**At-pickup**"**[^225]** | **rider reached pickup location** | **"In-progress"** |
| "Order-picked-up" | order picked up by rider | "In-progress" |
| "Out-for-delivery" | Out for delivery | "In-progress" |
| "**At-delivery**"**[^226]** | **rider reached delivery location** | **"In-progress"** |
| "Order-delivered" | delivered | "Completed" |
| "Cancelled" | cancelled | "Cancelled" |

2. **Intercity (P2H2P)**

| Fulfillment State | When to assign state | Order state |
| :---- | :---- | :---- |
| "Pending" | default fulfillment state | "Created" or "Accepted" |
| "Searching-for-Agent" | after RTS | "In-progress" |
| "Agent-assigned" | rider assigned | "In-progress" |
| **"Out-for-pickup"** | **out for pickup** | **"In-progress"** |
| **"Pickup-failed"** | **pickup attempted but failed** | **"In-progress"** |
| **"Pickup-rescheduled"** | **pickup rescheduled** | **"In-progress"** |
| "Order-picked-up" | picked up by rider | "In-progress" |
| **"In-transit"** | **at source hub** | **"In-progress"** |
| **"At-destination-hub"** | **at destination hub** | **"In-progress"** |
| "Out-for-delivery" | Out for delivery | "In-progress" |
| **"Delivery-failed"** | **delivery attempted but failed** | **"In-progress"** |
| **"Delivery-rescheduled"** | **delivery rescheduled** | **"In-progress"** |
| "Order-delivered" | delivered | "Completed" |
| "Cancelled" | cancelled | "Cancelled" |

3. **RTO**

| Fulfillment State | When to assign state | Order state |
| :---- | :---- | :---- |
| "RTO-Initiated" | When RTO has been initiated | Order state before RTO initiated |
| "RTO-Disposed"  | RTO terminal state when "return to origin" not required | "Cancelled" |
| "RTO-Delivered" | RTO terminal state when "return to origin" required  | "Cancelled" |

### Transaction level contract {#transaction-level-contract}

Overview

1. Purpose of the transaction level contract  between the participating NPs is explained [here](https://docs.google.com/document/d/1DPztkqv405W_t2o1LDNOL5t8Bv0K2j-s8UMG3v4Og0E/edit#) and includes the standard terms & conditions for the network participant(s);  
2. The codified digital contract includes a set of configurable terms defined by the seller and a set of static terms which will be defined by the LSP;  
3. A reference to these static terms will be a part of every online transaction between 2 participants;  
4. Reference clauses to serve as a guide for participants to draft their static terms for the codified digital contract can be found [here](https://docs.google.com/document/d/1hKImxDwZ-LUd1ln0D61KhhuQ9-Rqrzt2/edit#heading=h.9z4gdnelcvjh);

Proposed flow

1. Static terms will be version-controlled in GitHub [repo](https://github.com/ONDC-Official/static-terms):  
   1. All static terms will start with version 1.0.0 and subsequent changes will increment the version;  
   2. NP creates a PR for their static terms and ONDC will merge this PR and share the link with the NP:  
      1. NP forks the repository and creates a branch under the appropriate version, using the following naming convention:  
         * \<NP name\>\_role e.g. "Loadshare\_LSP", "Shadowfax\_LSP", etc.;  
         * role can have following values \- "LSP" (logistics service provider);  
      2. NP merges their static terms into this branch;  
      3. NP creates PR for these changes;

2. Static terms can be published by NP on multiple channels e.g. their website, ONDC portal, etc.

3. Communication & acceptance of static terms:

   1. **Communication by LSP**  
      1. LSP sends the following additional info as part of /on\_search:  
         * **current** static terms link with version:  
           * for the very 1st static terms, this will be a blank link;  
         * **new** static terms link with version:  
           * for the very 1st static terms, this will be the link for static terms in GitHub;  
         * effective date for new static terms;  
      2. Info in (i) above will be sent in /on\_search for full catalog pull for 24 hrs, after this version of static terms is published;  
      3. 24 hrs prior to expiry of effective date in (i) above, LSP resends additional info in (i) above;  
      4. If LSP wants to change the effective date, they can update the effective date in (i) above and repeat (ii) & (iii);  
      5. If LSP wants to change their static terms, they need to resubmit their PR with new version no;  
      6. It is recommended that the LSP should give at least 15 days of advance notice when they are changing their static terms;

   2. **Acceptance by logistics buyer NP (BNP)**  
      1. If BNP accepts the static terms in /on\_search, they need to send **accept**\="Y" in ACK for /on\_search;  
      2. If BNP doesn’t accept the static terms in /on\_search, they need to send **accept**\="N" in ACK for /on\_search;  
      3. If BNP doesn’t accept the static terms for a LSP, they should stop sending order requests to the LSP;  
      4. Static terms agreed to between the BNP & LSP will be a part of /confirm and /on\_confirm;

[^1]:  optional

[^2]:  optional

[^3]:  optional, enum \- "BPP", "BAP", "BG"

[^4]:  subscriber id of request initiator

[^5]:  unique identifier for request

[^6]:  timestamp in RFC3339 format

[^7]:  optional

[^8]:  enums \- "buyerApp", "sellerApp", "gateway"

[^9]:  search\_parameters signed using private key of request initiator:

[^10]:  [City code](https://docs.google.com/spreadsheets/d/12A_B-nDtvxyFh_FWDfp85ss2qpb65kZ7/edit?rtpof=true) should match the city for fulfillment start location;

[^11]:  enum:

[^12]:  store pickup timings;

[^13]:  days of week: 1 \- Monday till 7 \- Sunday;

[^14]:  list of future dated holidays;

[^15]:  order preparation time, indicates approx timing for logistics ready\_to\_ship;

[^16]:  **define store order pickup timing, for "days" above, here it means the store order pickup timings are from 1100 to 2100 from Monday to Sunday;**

[^17]:  **enum \- "Delivery", "Return";**

[^18]:  **minimum 6 digit decimal precision;**

[^19]:  **optional, only required if logistics buyer NP wants pickup authorization; if LSP doesn’t support authorization, they should not respond to the search request;**

[^20]:  enum \- "OTP";

[^21]:  **minimum 6 digit decimal precision;**

[^22]:  **optional, only required if logistics buyer NP wants delivery authorization; if LSP doesn’t support authorization, they should not respond to the search request;**

[^23]:  enum \- "OTP";

[^24]:  **enum \- "ON-ORDER" (prepaid), "ON-FULFILLMENT" (CoD), "POST-FULFILLMENT" (post fulfillment);**

[^25]:  **required for type \= "ON-FULFILLMENT";**

[^26]:  **enum \- "kilogram", "gram" (aligns with this [list](https://github.com/ONDC-Official/protocol-network-extension/blob/main/enums/retail/uom.json));**

[^27]:  **package dimensions \- mandatory for intercity shipments, optional for hyperlocal;**

[^28]:  **enum [here](https://github.com/ONDC-Official/protocol-network-extension/blob/main/enums/retail/dimension_unit.json);**

[^29]:  **use category from [here](https://docs.google.com/document/d/1brvcltG_DagZ3kGr1ZZQk4hG4tze3zvcxmGV4NMTzr8/edit#heading=h.w9zlp87xdha1) (e.g. Grocery, Fashion, etc);**

[^30]:  **optional, set to true when payload includes hazardous goods;**

[^31]:  name of logistics aggregator or logistics provider, as applicable;

[^32]:  category level TAT for S2D (ship-to-delivery), can be overridden by item-level TAT whenever there are multiple options for the same category (e.g. 30 min, 45 min, 60 min, etc.);

[^33]:  **refers to date (vis-a-vis Context.timestamp) for which this TAT is provided (same day for Immediate Delivery / SDD, next day for NDD, appropriate date for other categories);**

[^34]:  **average time to pickup (ISO8601 Duration);**

[^35]:  useful for P2P (hyperlocal);

[^36]:  **enum \- "mile", "kilometer", "meter";**

[^37]:  **shortest (preferably OSRM) distance between start & end locations;**

[^38]:  mandatory only for cases where shipment has to be dropped off at LSP location; **not required for P2P**;

[^39]:  indicates forward shipment item with quote;

[^40]:  type of shipment; enum \-  "P2P" (point-to-point) and "P2H2P" (point-to-hub-to-point). P2H2P fulfillments require different packaging and AWB no. This is required for merchants to decide on the packaging required for shipment;

[^41]:  **price is tax inclusive here, itemized in /on\_init;**

[^42]:  optional, item level TAT will override category-level TAT, if specified;

[^43]:  **refers to date (vis-a-vis Context.timestamp) for which this TAT is provided (same day for Immediate Delivery / SDD, next day for NDD, appropriate date for other categories) \- required for P2P;**

[^44]:  RTO quote linked to forward shipment quote through parent\_item\_id;

[^45]:  indicates RTO item with quote;

[^46]:  RTO quote (tax inclusive), will be added when RTO triggered through /on\_cancel;

[^47]:  optional, item level TAT will override category-level TAT, if specified;

[^48]:  **refers to date (vis-a-vis Context.timestamp) for which this TAT is provided (same day for Immediate Delivery / SDD, next day for NDD, appropriate date for other categories);**

[^49]:  enum \- "Y" (yes), "N" (no);

[^50]:  mandatory only if provider.locations was returned in /on\_search;

[^51]:  fulfillment\_id from /on\_search corresponding to appropriate type;

[^52]:  **name \+ building \+ locality \< 190 chars, name \!= locality;**

[^53]:  **name \+ building \+ locality \< 190 chars, name \!= locality;**

[^54]:  **optional**

[^55]:  **billing details of the marketplace (MSN or ISN retail seller app) for invoicing**;

[^56]:  name on the invoice;

[^57]:  address on invoice;

[^58]:  **required \- GST no for logistics buyer NP;**

[^59]:  **required;**

[^60]:  assumes LSP is collecting COD payment for the order;

[^61]:  optional \- only for ON-FULFILLMENT (CoD);

[^62]:  required for type ON-ORDER, ON-FULFILLMENT;

[^63]:  optional \- to be provided by LBNP if they’re not collecting payment;

[^64]:  enum \- "upi", "neft", "rtgs"; for "upi", upi\_address needs to be entered while for "neft", "rtgs", settlement\_bank\_account\_no, settlement\_ifsc\_code needs to be entered;

[^65]:  mandatory only if provider.locations was returned in /on\_search;

[^66]:  **check for rider availability, will be useful for Immediate Delivery;**

[^67]:  **enum \- delivery, tax, rto, diff;**

[^68]:  **ttl for quote;**

[^69]:  optional \- only for ON-FULFILLMENT (CoD);

[^70]:  enum \- "ON-FULFILLMENT" (CoD), "POST-FULFILLMENT" (post fulfillment billing), "ON-ORDER" (prepaid, typically thru wallet);

[^71]:  optional

[^72]:  fulfillment state (can be wildcard "\*", for all states);

[^73]:  reason code (can be wildcard "\*", for all reason codes);

[^74]:  cancellation fee, if any, for this state & reason code \- either percentage (of order value minus taxes, in the above format between "0.00" and "100.00") or amount needs to be provided (if both provided, minimum should be considered by LBNP); LSP can calculate based on what needs to be deducted e.g. delivery charges, etc;

[^75]:  list of reason codes delimited by "," (above shows RTO reason codes where buyer is not available);

[^76]:  in case of RTO where buyer is responsible, the forward & reverse shipment costs may be included in the cancellation fees;

[^77]:  logistics buyer NP has to accept LSP terms in /confirm. If logistics buyer doesn’t accept these terms, they should NACK /on\_init with error code 62501;

[^78]:  may be replaced with the URL of standard terms & conditions of network participant;

[^79]:  **alphanumeric, up to 32 characters;**

[^80]:  mandatory only if provider.locations was returned in /on\_search;

[^81]:  S2D TAT; if S2D TAT and / or average pickup time is different from what was quoted earlier in /on\_search, LSP can NACK /confirm with error code 60008;

[^82]:  AWB \# can be provided by logistics buyer NP (in /confirm or /update) or LSP (in /on\_confirm or /on\_update) and is mandatory only if item.descriptor.code \= "P2H2P". Can be 11-16 digits;

[^83]:  average pickup time;

[^84]:  **"code", "short\_desc" required if "ready\_to\_ship" \= yes in /confirm;**

[^85]:  **type of PCC: enum \- "2" \- merchant order no, "3" \- other pickup confirmation code, "4" \- OTP;**

[^86]:  **value of PCC:**

[^87]:  optional

[^88]:  (optional) reverse QC online checklist, may be provided for fulfillment type \= "Return";

[^89]:  optional \- may be provided after order is picked up;

[^90]:  **type of DCC: enum \- "1" \- OTP, "2" \- other DCC, "3" \- no delivery code;**

[^91]:  **value of DCC (required for code 1 & 2):**

[^92]:  optional

[^93]:  replaces "@ondc/org/order\_ready\_to\_ship";

[^94]:  enum \- "yes", "no";

[^95]:  whether RTO results in actual return to origin;

[^96]:  may be provided if fulfillment.type="Return";

[^97]:  optional

[^98]:  **use category from [here](https://docs.google.com/spreadsheets/d/1APAvavF_BNbTA89benAlGtv0GuFvpn2b6XXi4lSdTTw/edit#gid=1106356103);**

[^99]:  **unit weight, enum [here](https://github.com/ONDC-Official/protocol-network-extension/blob/main/enums/retail/uom.json);**

[^100]:  **total weight, enum [here](https://github.com/ONDC-Official/protocol-network-extension/blob/main/enums/retail/uom.json);**

[^101]:  **(mandatory for intercity, optional for hyperlocal) if weight and / or dimensions provided are different from what was initially provided (in /search), LSP can NACK /confirm with error code 60011;**

[^102]:  enum [here](https://github.com/ONDC-Official/protocol-network-extension/blob/main/enums/retail/dimension_unit.json);

[^103]:  logistics buyer NP must accept LSP terms. If not accepted, LSP can NACK /confirm with error code 65002;

[^104]:  enum \- "Created", "Accepted", "Cancelled", LSP may accept order after "ready\_to\_ship" from LBNP;

[^105]:  required only if in /confirm;

[^106]:  S2D TAT; If S2D TAT and / or average pickup time is different from what was quoted earlier in /on\_search, Logistics Buyer NP can NACK /on\_confirm with error code 62506;

[^107]:  required only if item.descriptor.code \= "P2H2P"; can be provided by logistics buyer (/confirm or /update) or LSP (/on\_confirm or /on\_update);

[^108]:  should be true if live tracking is enabled;

[^109]:  required if "ready\_to\_ship"="yes" in /confirm;

[^110]:  only if "ready\_to\_ship" \= "yes" in /confirm;

[^111]:  optional

[^112]:  **in response to RTS, LSP provides a pickup time slot. This time slot could be specific in case of "Immediate Delivery" or could be the entire day for SDD / NDD etc;**

[^113]:  may be provided if "ready\_to\_ship"="yes" in /confirm;

[^114]:  optional, may be provided for hyperlocal, inter-city delivery for last mile shipment;

[^115]:  optional

[^116]:  **weather conditions that buyer can be made aware of, may be provided for P2P (hyperlocal);**

[^117]:  may be provided if fulfillment.type="Return";

[^118]:  billing should be same as in /init

[^119]:  as in /confirm;

[^120]:  optional

[^121]:  **use category from [here](https://docs.google.com/spreadsheets/d/1APAvavF_BNbTA89benAlGtv0GuFvpn2b6XXi4lSdTTw/edit#gid=1106356103);**

[^122]:  **unit weight, enum \- "kilogram", "gram" (aligns with this [list](https://github.com/ONDC-Official/protocol-network-extension/blob/main/enums/retail/grocery/RET10-UOM.json));**

[^123]:  **total weight, enum \- "kilogram", "gram" (aligns with this [list](https://github.com/ONDC-Official/protocol-network-extension/blob/main/enums/retail/grocery/RET10-UOM.json));**

[^124]:  **if weight and / or dimensions provided are different from what was initially provided (in /search), LSP can NACK /confirm with error code 60011;**

[^125]:  **as proposed in /on\_init; if different from what was proposed in /on\_init, buyer NP can NACK with error code 62509;**

[^126]:  should match the corresponding timestamp in /confirm;

[^127]:  validation includes \- order object (items / quantity / quote / fulfillment) same as in /on\_init;

[^128]:  retry will have a cap for no of attempts over specific duration to be decided by BNP;

[^129]:  validation includes \- order object (id / items / quantity / quote) same as in /confirm;

[^130]:  retry will have a cap for no of attempts over specific duration to be decided by SNP;

[^131]:  required if item.descriptor.code \= "P2H2P" (may be provided in /confirm or /update by logistics buyer NP or /on\_confirm or /on\_update by LSP);

[^132]:  "code", "short\_desc" required if "ready\_to\_ship" \= yes;

[^133]:  type of PCC: enum \- "2" \- merchant order no, "3" \- other pickup confirmation code, "4" \- OTP;

[^134]:  value of PCC:

[^135]:  reverse QC online checklist required only for fulfillment type \= "Return" (if not in /confirm);

[^136]:  optional

[^137]:  enum \- "OTP", others TBD;

[^138]:  optional

[^139]:  type of DCC: enum \- "1" \- OTP, "2" \- other DCC, "3" \- no delivery code;

[^140]:  value of DCC:

[^141]:  optional, required for updating linked order details e.g. in case of part return or cancel. If not provided here, the linked order as provided in /confirm will hold;

[^142]:  only if weight changed from /confirm;

[^143]:  only if dimensions changed from /confirm;

[^144]:  required only if in /confirm;

[^145]:  **same item id as original item id since this is an additional weight-differential cost on top of the quoted price for the logistics service;**

[^146]:  **same item id as original item id since this is an tax on additional weight-differential cost for the logistics service;**

[^147]:  required for item.descriptor.code \= "P2H2P";

[^148]:  **proof of order picked up \- optional;**

[^149]:  **proof of order picked up \- optional;**

[^150]:  optional (may be provided for reverse QC)

[^151]:  optional, if available;

[^152]:  optional, if available;

[^153]:  optional

[^154]:  EBN \# & expiry date will be provided by LSP for inter-state shipments

[^155]:  **updated dimensions, only if there is difference in dimensions;**

[^156]:  **updated weight, only if there is difference in weight;**

[^157]:  **in case of updated dimensions and / or weight, the proof (images from sorter) can be added here;**

[^158]:  cancellation reason codes for [hyperlocal](https://docs.google.com/spreadsheets/d/1_qAtG6Bu2we3AP6OpXr4GVP3X-32v2xNRNSYQhhR6kA/edit?gid=610954815#gid=610954815) & [inter-city](https://docs.google.com/spreadsheets/d/1_qAtG6Bu2we3AP6OpXr4GVP3X-32v2xNRNSYQhhR6kA/edit?gid=956300231#gid=956300231);

[^159]:  **quote should reflect the updated value of logistics order (including weight differential cost & tax if applicable) which has to be paid by logistics buyer NP;**

[^160]:  required for P2H2P fulfillments;

[^161]:  optional

[^162]:  (optional) reference to issue related to cancel request raised in IGM, this will be added here only after issue resolution and returned in all post-order APIs like /on\_status, /on\_update, /on\_cancel;

[^163]:  pre-cancel state of fulfillment, i.e. state of fulfillment prior to cancellation;

[^164]:  **quote should reflect the updated value of logistics order (including weight differential cost & tax if applicable for forward shipment & RTO) which has to be paid by logistics buyer NP;**

[^165]:  additional RTO charges;

[^166]:  additional tax for RTO charges;

[^167]:  no of retries attempted;

[^168]:  fulfillment id of RTO fulfillment;

[^169]:  cancellation reason code from 2b [here](https://docs.google.com/document/d/1M-lZSduYMFKIk1V6d8QLt-j-16-rVzYVdPn0pmbkclk/edit#);

[^170]:  optional, may be provided in some cases e.g. if cancellation\_reason\_id is "008", sub\_reason\_id can include the pickup failure codes from [here](https://docs.google.com/spreadsheets/d/1_qAtG6Bu2we3AP6OpXr4GVP3X-32v2xNRNSYQhhR6kA/edit#gid=0);

[^171]:  bpp id of LSP;

[^172]:  (optional) reference to issue related to cancel request raised in IGM, this will be added here only after issue resolution and returned in all post-order APIs like /on\_status, /on\_update, /on\_cancel;

[^173]:  pre-cancel state of fulfillment, i.e. state of fulfillment prior to cancellation;

[^174]:  enum \- "RTO-Initiated", "RTO-Delivered", "RTO-Disposed";

[^175]:  matches the fulfillment id for which tracking is enabled. In case multiple fulfillments have tracking enabled, the id should match the fulfillment which has been picked up;

[^176]:  URL for non-hyperlocal tracking;

[^177]:  gps coordinates of rider;

[^178]:  time for which gps coordinates sent;

[^179]:  time when gps coordinates updated;

[^180]:  enum \- "active", "inactive"; if inactive, "location" will be empty. If tracking is enabled, status should become "active" when order is picked up;

[^181]:  path as a sequence of gps lat/lng coordinates;

[^182]:  order id being tracked;

[^183]:  whether tracking is by gps (point-to-point hyperlocal) or url (non-hyperlocal);

[^184]:  can be "tracking.location.gps", "tracking.url";

[^185]:  enum \- "live\_poll" (for attr="tracking.location.gps"), "deferred" (for attr="tracking.url");

[^186]:  enum \- "Created","Accepted","In-progress","Completed","Cancelled";

[^187]:  **required if order state is "Cancelled";**

[^188]:  required only if in /confirm;

[^189]:  if applicable;

[^190]:  if applicable;

[^191]:  if applicable;

[^192]:  if applicable;

[^193]:  enum for fulfillment state & mapping to order state for hyperlocal & inter-city is defined [here](#fulfillment-states-&-mapping-to-order-states);

[^194]:  pickup failure reason code [here](https://docs.google.com/spreadsheets/d/1_qAtG6Bu2we3AP6OpXr4GVP3X-32v2xNRNSYQhhR6kA/edit#gid=0), delivery failure reason code [here](https://docs.google.com/spreadsheets/d/1_qAtG6Bu2we3AP6OpXr4GVP3X-32v2xNRNSYQhhR6kA/edit#gid=1878676923);

[^195]:  required if fulfillment state code is "Order-picked-up", "Out-for-delivery", "Order-delivered"; after order is picked up, time range for delivery (end.time.range) may be updated, as applicable;

[^196]:  optional

[^197]:  optional

[^198]:  images will have following entries in this order \- shipping label (intercity) to be provided by LSP, proof of pickup (only after order picked up) \- optional;

[^199]:  (optional) if applicable, i.e. only in cases of reverse QC;

[^200]:  **optional, only for pickup authorization. If authorization is enabled by seller NP, LSP needs to provide authorization type & code, while updating fulfillment state to "Order-picked-up". Seller NP should verify this and for invalid / expired authorization, NACK the request with the following error codes:**

[^201]:  enum \- "OTP", others TBD;

[^202]:  required if fulfillment state code is "Order-delivered";

[^203]:  optional

[^204]:  optional

[^205]:  proof of delivery \- optional;

[^206]:  **optional, only for delivery authorization. If authorization is enabled by seller NP, LSP needs to provide authorization type & code, while updating fulfillment state to "Order-delivered". Seller NP should verify this and for invalid / expired authorization, NACK the request with the following error codes:**

[^207]:  enum \- "OTP", others TBD;

[^208]:  optional, may be provided for inter-city shipments for last mile delivery

[^209]:  optional

[^210]:  may be provided if fulfillment.type="Return" for the following states \- "Pickup-failed" or "Order-picked-up" or "Delivery-failed" or "Order-delivered";

[^211]:  enum \- "Y" (yes) or "N" (no);

[^212]:  audit trail for pickup / delivery delay, including attempt made for pickup / delivery:

[^213]:  enum \- "Pickup-failed", "Order-picked-up", "Delivery-failed", "Order-delivered";

[^214]:  reason codes for [pickup delay / failure](https://docs.google.com/spreadsheets/d/1_qAtG6Bu2we3AP6OpXr4GVP3X-32v2xNRNSYQhhR6kA/edit#gid=0), [delivery delay / failure](https://docs.google.com/spreadsheets/d/1_qAtG6Bu2we3AP6OpXr4GVP3X-32v2xNRNSYQhhR6kA/edit#gid=1878676923);

[^215]:  (optional) tracking attributes for fulfillment, may be provided, on or before fulfillment state of "Agent-assigned", if tracking \= "true";

[^216]:  whether tracking by GPS coordinates enabled;

[^217]:  whether tracking by GPS coordinates enabled;

[^218]:  if tracking by URL enabled, corresponding URL for tracking;

[^219]:  required if fulfillment state code is "RTO-Initiated", "RTO-Delivered", "RTO-Disposed";

[^220]:  optional

[^221]:  **required for payment on delivery orders when order state is "Completed"**;

[^222]:  **updated dimensions, only if applicable;**

[^223]:  **updated weight, only if applicable;**

[^224]:  **in case of updated dimensions and / or weight, the proof (images from sorter) can be added here, only if applicable;**

[^225]:  optional, to be set when rider reaches perimeter of pickup location, as specified by LSP;

[^226]:  optional, to be set when rider reaches perimeter of delivery location, as specified by LSP;