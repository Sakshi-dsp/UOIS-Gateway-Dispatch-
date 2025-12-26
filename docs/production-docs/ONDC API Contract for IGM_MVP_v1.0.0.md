**Revision History**

| Version | Date | Changes |
| :---- | :---- | :---- |
| 2.0.0 | 27th Dec 2024 | This version has been deprecated.  Follow the v2.0.0, features and flows available [here](https://docs.google.com/document/d/1CBTOuFP4o6KqwYVPlN6H9ArUvygMjFiII-gSv4WXk4s/edit?usp=sharing) and specs available [here](https://ondc-official.github.io/ONDC-NTS-Specifications) (version draft-IGM-2.0.0).  |
|  1.0.0 | 7th Oct 2024 | Updated enums for resolution.action\_triggered for handling payment/ RSF related complaints Updated footnotes for addressing applicability of the specs across all use cases |
|  | 13th Feb 2024 | Added clarification footnote for context object being used for various domain |
|  | 28th Apr 2023 | action\_triggered \- enums added \- NO-ACTION, CANCEL issue.issue\_action changed to issue\_actions source.issue\_source\_type changed to type odr.about\_info changed to long\_desc odr.short\_desc added rating.rating\_value changed to value issue\_status.message.id changed to issue\_id respondentemail removed pos\_id removed from respondent\_info object respondentcontact changed to contact respondentchatlink changed to chat\_link respondentFaqs changed to faqs faqs made into an array additional\_sources made into an array resolution.resolution changed to short\_desc resolution.resolution\_remarks changed to long\_desc dispute\_resolution\_remarks changed to odr\_remarks resolution\_action changed to action, removed from API contract complainant\_actions.remarks changed to short\_desc respondent\_actions.remarks changed to short\_desc Addition of notes on reflection of retail order updates done through core retail APIs based on the resolution provided Swagger hub API specs updated |
|  | 26th Apr 2023 | Added illustrative flow 5b, 9b in case the buyer is not satisfied with the resolution provided for the escalation flow Added 4b where the seller app provides the action trail updates to the buyer app Added 10a where the seller app cascaded the complaint closure to the logistics provider "name" attribute removed from resolution\_provider.respondent\_info.resolution\_support.respondent\_contact object |
|  | 20th Apr 2023 | "pos\_id" renamed to "merchant\_order\_id" |
|  | 18th Apr 2023 | Domain code updated as "ONDC:RET10" based on the new codes as part of the v1.2.0 "quantity" added in the items object signifying the quantity for which the complaint has been raised "refund\_amount" added in the resolution object signifying the amount to be refunded for the complaint Changed ttl of API calls to PT30S similar to it being used in the transaction APIs, TAT management of complaints will take place through the attributes of expected\_responde\_time and expected\_resolution\_time |
| 0.1.1 | 10th Apr 2023 | Added different payloads for seller app providing resolution to the buyer app for Scenario 2 (2a & 2b) Enums for sub category added |
| 0.1.0 | 27th Mar 2023 | Initial draft |

## Overview {#overview}

The implementation for Issue and Grievance Management framework will be done in a phased manner. The proposed phase wise implementation has been documented [here](https://docs.google.com/document/d/1BD3hZCy_KHq9V6EMq5MzfFsv9CSFNl3Tszm5u5s0fpg/edit?usp=sharing).

The MVP Phase 1 implementation of IGM will cover only the following scenarios: [^1]

1. Buyer raising a complaint related to an item in an order  
2. Buyer raising a complaint related to a fulfillment

The API contract (along with the specifications published on GitHub and Swagger Hub) covers all the enums and optional attributes that are required for various flows.

This document covers the sample payloads with only the mandatory attributes for the complete API flows for the above two scenarios.

Please refer to the following:

[IGM process flow](https://docs.google.com/document/d/135OCfsi5jQ7wh4H_LOoMxb0T0ZrWDYy4LTBvpYS6k_w/edit)

[Swagger link](https://app.swaggerhub.com/apis/ONDC/ONDC-Protocol-IGM/1.0.0)

[Developer guide](https://ondc-official.github.io/ONDC-NTS-Specifications/?branch=release-IGM-1.0.0) (only for illustrative payloads)

[Overview](#overview)

[Reflection of resolution provided through IGM](#reflection-of-resolution-provided-through-igm)

[API Contract Payloads](#api-contract-payloads)

[Scenario 1: Complaints related to an item](#scenario-1:-complaints-related-to-an-item-category)

[1\. Buyer app sending complaint details to retail seller app](#1.-buyer-app-sending-complaint-details-to-seller-app)

[2\. Seller app responding marking the respondent action as processing signifying they have started processing the issue](#2.-seller-app-responding-marking-the-respondent-action-as-processing-signifying-they-have-started-processing-the-issue)

[3\. Buyer app requesting status update on the complaint](#3.-buyer-app-requesting-status-update-on-the-complaint)

[4\. Seller app providing status update on the complaint](#4.-seller-app-providing-status-update-on-the-complaint)

[5a. Buyer app closing the complaint as closed by the buyer](#5a.-buyer-app-closing-the-complaint-as-closed-by-the-buyer)

[5b. Buyer app escalating the complaint as escalated by the buyer](#5b.-buyer-app-escalating-the-complaint-as-escalated-by-the-buyer)

[Scenario 2: Complaints related to a fulfillment](#scenario-2:-complaints-related-to-a-fulfillment---for-cascaded-transactions)

[1\. Buyer app sending complaint details to retail seller app](#1.-buyer-app-sending-complaint-details-to-retail-seller-app)

[2\. Retail seller app providing update on the complaint to the buyer app](#2.-retail-seller-app-providing-update-on-the-complaint-to-the-buyer-app)

[3\. Retail seller app sending complaint details to logistic seller app](#context/timestamp--should-be-greater-than-the-or-equal-to-the-message/issue/updated_at-.-3.-retail-seller-app-sending-complaint-details-to-logistic-seller-app)

[4a. Logistics seller app responding marking the respondent action as processing signifying they have started processing the issue](#4a.-logistics-seller-app-responding-marking-the-respondent-action-as-processing-signifying-they-have-started-processing-the-issue)

[4b. Seller App providing the updated trail of actions to the buyer app](#4b.-seller-app-providing-the-updated-trail-of-actions-to-the-buyer-app)

[5\. Buyer app requesting status update on the complaint from the retail seller app](#5.-buyer-app-requesting-status-update-on-the-complaint-from-the-retail-seller-app)

[6\. Retail seller app requesting status update on the complaint from the logistics seller app](#w6.-retail-seller-app-requesting-status-update-on-the-complaint-from-the-logistics-seller-app)

[7\. Logistics seller app providing status update on the complaint to the retail seller app](#7.-logistics-seller-app-providing-status-update-on-the-complaint-to-the-retail-seller-app)

[8\. Retail Seller app providing status update on the complaint to the buyer app](#8.-retail-seller-app-providing-status-update-on-the-complaint-to-the-buyer-app)

[Scenario 8a. Retail seller app cascading the resolution provided by the logistics provider to the buyer app](#scenario-8a.-retail-seller-app-cascading-the-resolution-provided-by-the-logistics-provider-to-the-buyer-app)

[Scenario 8b. Retail seller app providing the resolution provider to the buyer app](#scenario-8b.-retail-seller-app-providing-the-resolution-provider-to-the-buyer-app)

[9a. Buyer app closing the complaint as closed by the buyer](#9a.-buyer-app-closing-the-complaint-as-closed-by-the-buyer)

[9b. Buyer app escalating the complaint as escalated by the buyer](#9b.-buyer-app-escalating-the-complaint-as-escalated-by-the-buyer)

[10a. Seller app closing the complaint to the logistics provider](#10a.-seller-app-closing-the-complaint-to-the-logistics-provider)

[FAQs](#faqs)

## Reflection of resolution provided through IGM {#reflection-of-resolution-provided-through-igm}

IGM APIs will be used for handling complaints on the network between NPs to get a resolution for the complaint raised. The implementation of the resolution provided (whether it be refund/ replacement/ cancel), that is, the changes required on the order object will continue to be through the retail APIs that are the /update or /on\_update APIs.  
There are certain use cases where /update API can be used directly. For example, for cancellation requests for cancellable items, a request for cancelling the item can still go through /update API. But for items that are non-cancellable, the cancellation request should ideally go the IGM route.

As part of the IGM framework for resolution of complaints using the IGM APIs, the resolution provider will be providing a resolution for the complaint raised. In the current context where buyer is raising an issue on the buyer app and the resolution provider can either be the retail seller app or the logistics provider, there can be primarily of three types resolutions provided:

* Partial or full refund (with or without return)  
  * In case the resolution provided is a refund for the relevant items, the initiation of the refunds has to be done immediately, with the order object getting updated with the /on\_update call with requisite information, calculation and reference to the issue.id for which the refund is being initiated  
  * In case the buyer is not satisfied with the refund, option to escalate may be utilised by buyer  
* Replacement  
  * In case the resolution provided is a replacement, the buyer is given this resolution proposal. The buyer can either accept or reject the resolution  
  * In case the buyer accepts the resolution, the replacement is triggered with the /on\_update API with requisite information and reference to the issue.id for which the replacement is being initiated  
  * In case the buyer rejects the replacement resolution, a “need\_more\_info” flow is initiated from the buyer side, where they may share their relevant request (provision of a refund instead of the replacement) and accordingly a new resolution may be proposed by the resolution provider  
  * In case there is no action by the buyer for a period of time (X days), the replacement doesn’t take place and the complaint is closed  
* No action  
  * In case no action is provided, the buyer still has an opportunity to escalate the complaint

The closure of the issue remains a buyer (complainant) action. Should the buyer not take any action in a pre-defined time period (as defined part of [IGM policy](https://ondc-static-website-media.s3.ap-south-1.amazonaws.com/ondc-website-media/downloads/governance-and-policies/CHAPTER+%5B6%5D+Issue+and+Grievance+Management+Policy.pdf)) after receiving a resolution, the complaint will be marked as closed.

## API Contract Payloads[^2] {#api-contract-payloads}

## Scenario 1: Complaints related to an item[^3] category {#scenario-1:-complaints-related-to-an-item-category}

### 1\. Buyer app sending complaint details to seller app {#1.-buyer-app-sending-complaint-details-to-seller-app}

/issue  
{  
  "context[^4]":  
  {  
    "domain": "**ONDC:RET10[^5]**",  
    "country": "IND",  
    "city": "std:080",  
    "action": "issue",         
    "core\_version": "1.0.0",  
    "bap\_id": "buyerapp.com",  
    "bap\_uri": "https://buyerapp.com/ondc",  
    "bpp\_id": "sellerapp.com",  
    "bpp\_uri": "https://sellerapp.com/ondc",  
    "transaction\_id": "T1[^6]",  
    "message\_id": "M1",  
    "timestamp": "2023-01-15T10:00:00.469Z",  
    "ttl": "PT30S"  
  },  
  "message":   
  {  
    "issue":  
    {  
      "id": "I1",  
      "category[^7]": "ITEM",  
      "sub\_category[^8]": "ITM04",  
      "complainant\_info":  
      {  
        "person":  
        {  
          "name": "Sam Manuel"  
        },  
        "contact":  
        {  
          "phone": "9879879870",  
          "email[^9]": "sam@yahoo.com"  
        }  
      },  
      "order\_details":  
      {  
        "id": "4597f703-e84f-431e-a96a-d147cfa142f9",  
        "state[^10]": "Completed",  
        "items[^11]":   
        \[  
          {  
            "id": "18275-ONDC-1-9",  
            **"quantity[^12]": 1**  
          }  
        \],  
        "fulfillments[^13]":  
        \[   
          {  
            "id": "Fulfillment1",  
            "state[^14]": "Order-delivered"  
          }      
        \],  
        "provider\_id": "P1"  
      },  
      "description":   
      {  
        "short\_desc": "Issue with product quality",  
        "long\_desc": "product quality is not correct. facing issues while using the product",  
        "additional\_desc[^15]":   
        {  
          "url": "https://buyerapp.com/additonal-details/desc.txt",  
          "content\_type": "text/plain"  
        },  
        "images[^16]": \[  
              "http://buyerapp.com/addtional-details/img1.png",  
              "http://buyerapp.com/addtional-details/img2.png"  
            \]  
      },  
      "source": {  
        "network\_participant\_id": "buyerapp.com/ondc",  
        "type": "CONSUMER[^17]"  
      },  
      "expected\_response\_time[^18]": {  
        "duration": "PT2H"  
      },  
      "expected\_resolution\_time": {  
        "duration": "P1D"  
      },  
      "status": "OPEN[^19]",  
      "issue\_type": "ISSUE[^20]",  
      "issue\_actions":  
      {  
        "complainant\_actions":  
        \[  
        {  
          "complainant\_action": "OPEN[^21]",  
          "short\_desc": "Complaint created",  
          "updated\_at": "2023-01-15T10:00:00.469Z",  
          "updated\_by[^22]":  
          {  
            "org":  
            {  
              "name": "buyerapp.com::ONDC:RET10[^23]"  
            },  
            "contact":  
            {  
              "phone": "9450394039",  
              "email": "buyerapp@interface.com"  
            },  
            "person":  
            {  
              "name": "John Doe"  
            }  
          }  
        }  
        \]  
      },  
      "created\_at": "2023-01-15T10:00:00.469Z",  
      "updated\_at": "2023-01-15T10:00:00.469Z"  
    }  
  }  
}

### 

### 2\. Seller app responding marking the respondent action as processing signifying they have started processing the issue {#2.-seller-app-responding-marking-the-respondent-action-as-processing-signifying-they-have-started-processing-the-issue}

/on\_issue  
{  
  "context":  
  {  
    "domain": "ONDC:RET10",  
    "country": "IND",  
    "city": "std:080",  
    "action": "on\_issue",  
    "core\_version": "1.0.0",  
    "bap\_id": "buyerapp.com",  
    "bap\_uri": "https://buyerapp.com/ondc",  
    "bpp\_id": "sellerapp.com",  
    "bpp\_uri": "https://sellerapp.com/ondc",  
    "transaction\_id": "T1",  
    "message\_id": "M1",  
    "timestamp": "2023-01-15T10:10:00.142Z"  
  },  
  "message":   
  {  
    "issue":  
    {  
      "id": "I1",  
      "issue\_actions":  
      {  
        "respondent\_actions":  
        \[  
        {  
          "respondent\_action": "PROCESSING[^24]",  
          "short\_desc": "Complaint is being processed",  
          "updated\_at": "2023-01-15T10:10:00.142Z",  
          "updated\_by[^25]":  
          {  
            "org":  
            {  
              "name": "sellerapp.com/ondc::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9450394140",  
              "email": "respondentapp@respond.com"  
            },  
            "person":  
            {  
              "name": "Jane Doe"  
            }  
          },  
          "cascaded\_level":1  
        }  
        \]  
      },  
      "created\_at": "2023-01-15T10:00:00.469Z",  
      "updated\_at": "2023-01-15T10:10:00.142Z"  
    }  
  }  
}

### 3\. Buyer app requesting status update on the complaint {#3.-buyer-app-requesting-status-update-on-the-complaint}

/issue\_status  
{  
  "context":  
  {  
    "domain": "ONDC:RET10",  
    "country": "IND",  
    "city": "std:080",  
    "action": "issue\_status",  
    "core\_version": "1.0.0",  
    "bap\_id": "buyerapp.com",  
    "bap\_uri": "https://buyerapp.com/ondc",  
    "bpp\_id": "sellerapp.com",  
    "bpp\_uri": "https://sellerapp.com/ondc",  
    "transaction\_id": "T1",  
    "message\_id": "M2",  
    "timestamp": "2023-01-15T10:30:00.469Z",  
    "ttl": "PT30S"  
  },  
  "message":   
  {  
    "issue\_id": "I1"  
  }  
}

### 4\. Seller app providing status update on the complaint {#4.-seller-app-providing-status-update-on-the-complaint}

/on\_issue\_status  
{  
  "context":   
  {  
    "domain": "ONDC:RET10",  
    "country": "IND",  
    "city": "std:080",  
    "action": "on\_issue\_status",  
    "core\_version": "1.0.0",  
    "bap\_id": "buyerapp.com",  
    "bap\_uri": "https://buyerapp.com/ondc",  
    "bpp\_id": "sellerapp.com",  
    "bpp\_uri": "https://sellerapp.com/ondc",  
    "transaction\_id": "T1",  
    "message\_id": "M2",  
    "timestamp": "2023-01-15T10:31:00.523Z"  
  },  
  "message":  
  {  
    "issue":  
    {  
      "id": "I1",  
      "issue\_actions":  
      {  
        "respondent\_actions":  
        \[  
        {  
          "respondent\_action": "PROCESSING",  
          "short\_desc": "Complaint is being processed",  
          "updated\_at": "2023-01-15T10:10:00.142Z",  
          "updated\_by[^26]":  
          {  
            "org":  
            {  
              "name": "sellerapp.com::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9450394140",  
              "email": "respondentapp@respond.com"  
            },  
            "person":  
            {  
              "name": "Jane Doe"  
            }  
          },  
          "cascaded\_level[^27]":1  
        },  
        {  
          "respondent\_action": "RESOLVED",  
          "short\_desc": "Complaint resolved",  
          "updated\_at": "2023-01-15T10:31:00.523Z",  
          "updated\_by":  
          {  
            "org":  
            {  
              "name": "sellerapp.com::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9450394140",  
              "email": "respondentapp@respond.com"  
            },  
            "person":  
            {  
              "name": "Jane Doe"  
            }  
          },  
          "cascaded\_level":1  
        }  
        \]  
      },  
      "created\_at": "2023-01-15T10:00:00.469Z",  
      "updated\_at": "2023-01-15T10:31:00.523Z",  
      "resolution\_provider[^28]":  
      {  
        "respondent\_info":  
        {  
          "type": "TRANSACTION-COUNTERPARTY-NP[^29]",  
          "organization":  
          {  
            "org":  
            {  
              "name": "sellerapp.com::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9059304940",  
              "email": "email@resolutionproviderorg.com"  
            },  
            "person":  
            {  
              "name": "resolution provider org contact person name"  
            }  
          },  
          "resolution\_support":  
          {  
            "chat\_link[^30]": "http://chat-link/respondent",  
            "contact":  
            {  
              "phone": "9949595059",  
              "email": "respondantemail@resolutionprovider.com"  
            },  
            "gros[^31]":  
            \[  
            {  
              "person":   
              {  
                "name": "Sam D"  
              },  
              "contact":  
              {  
                "phone": "9605960796",  
                "email": "email@gro.com"  
              },  
              "gro\_type": "TRANSACTION-COUNTERPARTY-NP-GRO[^32]"  
            }  
            \]  
          }  
        }  
      },  
      "resolution[^33]":   
      {  
        "short\_desc": "Refund to be initiated",  
        "long\_desc[^34]": "For this complaint, refund is to be initiated",  
        "action\_triggered": "REFUND[^35]",  
        **"refund\_amount[^36]": "100"**  
      }  
    }  
  }  
}

Note: 

* Once a resolution is provided by a resolution provider, the buyer (complainant) can either close the complaint or escalate it.

### 

### 5a. Buyer app closing the complaint as closed by the buyer {#5a.-buyer-app-closing-the-complaint-as-closed-by-the-buyer}

/issue  
{  
  "context":   
  {  
    "domain": "ONDC:RET10",  
    "country": "IND",  
    "city": "std:080",  
    "action": "issue",  
    "core\_version": "1.0.0",  
    "bap\_id": "abc.interfacingapp.com",  
    "bap\_uri": "https://abc.buyerapp.com/ondc",  
    "bpp\_id": "xyz.sellerapp.com",  
    "bpp\_uri": "https://xyz.sellerapp.com/ondc",  
    "transaction\_id": "T1",  
    "message\_id": "M5",  
    "timestamp": "2023-01-15T12:43:98.982Z",  
    "ttl": "PT30S"  
  },  
  "message":  
  {  
    "issue":   
    {  
      "id": "I1",  
      "status": "CLOSED",  
      "issue\_actions":  
      {  
        "complainant\_actions":  
        \[  
        {  
          "complainant\_action": "OPEN",  
          "short\_desc": "Complaint created",  
          "updated\_at": "2023-01-15T10:00:00.469Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "abc.buyerapp.com/ondc::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9450394039",  
              "email": "interfacingapp@interface.com"  
            },  
            "person":  
            {  
              "name": "John Doe"  
            }  
          }  
        },  
        {  
          "complainant\_action": "CLOSE",  
          "short\_desc": "Complaint closed",  
          "updated\_at": "2023-01-15T12:43:98.982Z",  
          "updated\_by":   
          {  
            "org":  
            {  
             "name": "abc.buyerapp.com/ondc::ONDC:RET10"  
             },  
            "contact":  
            {  
              "phone": "9450394039",  
              "email": "interfacingapp@interface.com"  
            },  
            "person":  
            {  
              "name": "John Doe"  
            }  
          }  
        }  
        \]  
      },  
      "rating[^37]": "THUMBS-UP",  
      "created\_at": "2023-01-15T10:00:00.469Z",  
      "updated\_at": "2023-01-15T10:41:30.723Z"  
    }  
  }  
}

### 5b. Buyer app escalating the complaint as escalated by the buyer [^38] {#5b.-buyer-app-escalating-the-complaint-as-escalated-by-the-buyer}

/issue  
{  
  "context":   
  {  
    "domain": "ONDC:RET10",  
    "country": "IND",  
    "city": "std:080",  
    "action": "issue",  
    "core\_version": "1.0.0",  
    "bap\_id": "abc.interfacingapp.com",  
    "bap\_uri": "https://abc.buyerapp.com/ondc",  
    "bpp\_id": "xyz.sellerapp.com",  
    "bpp\_uri": "https://xyz.sellerapp.com/ondc",  
    "transaction\_id": "T1",  
    "message\_id": "M5",  
    "timestamp": "2023-01-15T12:43:98.982Z",  
    "ttl": "PT30S"  
  },  
  "message":  
  {  
    "issue":   
    {  
      "id": "I1",  
      "status": "OPEN",  
      "issue\_type": "GRIEVANCE",  
      "issue\_actions":  
      {  
        "complainant\_actions":  
        \[  
        {  
          "complainant\_action": "OPEN",  
          "short\_desc": "Complaint created",  
          "updated\_at": "2023-01-15T10:00:00.469Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "abc.buyerapp.com/ondc::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9450394039",  
              "email": "interfacingapp@interface.com"  
            },  
            "person":  
            {  
              "name": "John Doe"  
            }  
          }  
        },  
        {  
          "complainant\_action": "ESCALATE",  
          "short\_desc": "Not satisfied with the resolution provided",  
          "updated\_at": "2023-01-15T12:43:98.982Z",  
          "updated\_by":   
          {  
            "org":  
            {  
             "name": "abc.buyerapp.com/ondc::ONDC:RET10"  
             },  
            "contact":  
            {  
              "phone": "9450394039",  
              "email": "interfacingapp@interface.com"  
            },  
            "person":  
            {  
              "name": "John Doe"  
            }  
          }  
        }  
        \]  
      },  
      "created\_at": "2023-01-15T10:00:00.469Z",  
      "updated\_at": "2023-01-15T10:41:30.723Z"  
    }  
  }  
}

Note:

* Escalation flows is not planned as part of the MVP for IGM implementation, though the APIs work in a similar manner, GROs communicate amongst themselves in case the complaint is escalated

## Scenario 2: Complaints related to a fulfillment \- for cascaded transactions {#scenario-2:-complaints-related-to-a-fulfillment---for-cascaded-transactions}

### 1\. Buyer app sending complaint details to retail seller app {#1.-buyer-app-sending-complaint-details-to-retail-seller-app}

/issue  
{  
  "context":   
  {  
    "domain": "ONDC:RET10",  
    "country": "IND",  
    "city": "std:080",  
    "action": "issue",  
    "core\_version": "1.0.0",  
    "bap\_id": "abc.interfacingapp.com",  
    "bap\_uri": "https://abc.buyerapp.com/ondc",  
    "bpp\_id": "xyz.sellerapp.com",  
    "bpp\_uri": "https://xyz.sellerapp.com/ondc",  
    "transaction\_id": "T1",  
    "message\_id": "M1",  
    "timestamp": "2023-01-15T10:00:00.469Z",  
    "ttl": "PT30S"  
  },  
  "message":   
  {  
    "issue":  
    {[^39]  
      "id": "I2",  
      "category": "FULFILLMENT",  
      "sub\_category": "FLM01",  
      "complainant\_info":  
      {  
        "person":  
        {  
          "name": "Sam Manuel"  
        },  
        "contact": {  
          "phone": "9879879870",  
          "email": "sam@yahoo.com"  
        }  
      },  
      "order\_details":  
     {  
        "id": "4597f703-e84f-431e-a96a-d147cfa142f9",  
        "state": "Completed",  
        "items":  
        \[  
          {  
            "id": "18275-ONDC-1-9",  
            "quantity": 1  
          }  
        \],  
        "fulfillments":  
        \[  
        {  
          "id": "Fulfillment1",  
          "state": "Order delivered"  
        }  
        \],  
        "provider\_id": "P1"  
      },  
      "description": {  
        "short\_desc": "Issue with product delivery",  
        "long\_desc": "product delivery is not correct. It was delayed by 5 days.",  
        "additional\_desc": {  
          "url": "https://interfac-app/igm/additonal-desc/user/desc.txt",  
          "content\_type": "text/plain"  
        },  
        "images": \[  
          "http://interfacing.app/addtional-details/img1.png",  
          "http://interfacing.app/addtional-details/img2.png"  
        \]  
      },  
      "source": {  
        "network\_participant\_id": "abc.buyerapp.com",  
        "type": "CONSUMER"  
      },  
      "expected\_response\_time": {  
        "duration": "PT2H"  
      },  
      "expected\_resolution\_time": {  
        "duration": "P1D"  
      },  
      "status": "OPEN",  
      "issue\_type": "ISSUE",  
      "issue\_actions":  
      {  
        "complainant\_actions":  
        \[  
        {  
          "complainant\_action": "OPEN",  
          "short\_desc": "Complaint created",  
          "updated\_at": "2023-01-15T10:00:00.469Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "abc.buyerapp.com::ONDC:RET10"  
            },  
            "contact"  
            {  
              "phone": "9450394039",  
              "email": "interfacingapp@interface.com"  
            },  
            "person":  
            {  
              "name": "John Doe"  
            }  
          }  
        }  
        \]  
      }  
      "created\_at": "2023-01-15T10:00:00.469Z",  
      "updated\_at": "2023-01-15T10:00:00.469Z"  
    }  
  }  
}

### 2\. Retail seller app providing update on the complaint to the buyer app {#2.-retail-seller-app-providing-update-on-the-complaint-to-the-buyer-app}

/on\_issue  
{  
  "context":  
  {  
    "domain": "ONDC:RET10",  
    "country": "IND",  
    "city": "std:080",  
    "action": "on\_issue",  
    "core\_version": "1.0.0",  
    "bap\_id": "abc.interfacingapp.com",  
    "bap\_uri": "https://abc.buyerapp.com/ondc",  
    "bpp\_id": "xyz.sellerapp.com",  
    "bpp\_uri": "https://xyz.sellerapp.com/ondc",  
    "transaction\_id": "T1",  
    "message\_id": "M1",  
    "timestamp": "2023-01-15T10:30:20.162Z"  
  },  
  "message":  
  {  
    "issue":   
    {  
      "id": "I2",  
      "issue\_actions":   
      {  
        "respondent\_actions":  
        \[  
        {  
          "respondent\_action": "PROCESSING",  
          "short\_desc": "Complaint is being processed",  
          "updated\_at": "2023-01-15T10:04:01.812Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "xyz.sellerapp.com::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9960394039",  
              "email": "transactioncounterpartyapp@tcapp.com"  
            },  
            "person":  
            {  
              "name": "James Doe"  
            }  
          },  
          "cascaded\_level":1  
        }  
        \]  
      },  
      "created\_at": "2023-01-15T10:00:00.469Z",  
      "updated\_at": "2023-01-15T10:00:00.469Z"  
    }  
  }  
}

### context/timestamp- should be greater than the or equal to the message/issue/updated\_at . 3\. Retail seller app sending complaint details to logistic seller app {#context/timestamp--should-be-greater-than-the-or-equal-to-the-message/issue/updated_at-.-3.-retail-seller-app-sending-complaint-details-to-logistic-seller-app}

/issue  
{  
  "context": {  
    "domain": "nic2004:60232",  
    "country": "IND",  
    "city": "std:080",  
    "action": "issue",  
    "core\_version": "1.0.0",  
    "bap\_id": "xyz.sellerapp.com",  
    "bap\_uri": "https://xyz.sellerapp.counterpary.com",  
    "bpp\_id": "lmn.logisticssellerapp.com",  
    "bpp\_uri": "https://lmn.logisticssellerapp.counterpary.com",  
    "transaction\_id": "T2",  
    "message\_id": "M2",  
    "timestamp": "2023-01-15T10:05:00.267Z",  
    "ttl": "PT30S"  
  },  
  "message":   
  {  
    "issue":  
    {  
      "id": "I2",  
      "category": "FULFILLMENT",  
      "sub\_category": "FLM01",  
      "complainant\_info":  
      {  
        "person":  
        {  
          "name": "Sam Manuel"  
        },  
        "contact": {  
          "phone": "9879879870",  
          "email": "sam@yahoo.com"  
        }  
      },  
      "order\_details":   
      {  
        "id": "4597f703-e84f-431e-a96a-d147cfa142f9",  
        "state": "Completed",  
        "items":  
        \[  
          {  
            "id": "18275-ONDC-1-9",  
            "quantity": 1  
          }  
        \],  
        "fulfillments":  
        \[  
        {  
          "id": "Fulfillment1",  
          "state": "Order-delivered"  
        }  
        \],  
        "provider\_id": "P1",  
        "merchant\_order\_id[^40]": "101"  
      },  
      "description": {  
        "short\_desc": "Issue with product delivery",  
        "long\_desc": "product delivery is not correct. It was delayed by 5 days.",  
        "additional\_desc": {  
          "url": "https://interfac-app/igm/additonal-desc/user/desc.txt",  
          "content\_type": "text/plain"  
        },  
        "images": \[  
          "http://interfacing.app/addtional-details/img1.png",  
          "http://interfacing.app/addtional-details/img2.png"  
        \]  
      },  
      "source": {  
        "network\_participant\_id": "abc.interfacingapp.com",  
        "type": "CONSUMER"  
      },  
      "expected\_response\_time": {  
        "duration": "PT2H"  
      },  
      "expected\_resolution\_time": {  
        "duration": "P1D"  
      },  
      "status": "OPEN",  
      "issue\_type": "ISSUE",  
      "issue\_actions[^41]":  
      {  
        "complainant\_actions":  
        \[  
        {  
          "complainant\_action": "OPEN",  
          "short\_desc": "Complaint created",  
          "updated\_at": "2023-01-15T10:00:00.469Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "abc.buyerapp.com::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9450394039",  
              "email": "interfacingapp@interface.com"  
            },  
            "person":  
            {  
              "name": "John Doe"  
            }  
          }  
        }  
        \],  
        "respondent\_actions":  
        \[  
        {  
          "respondent\_action": "PROCESSING",  
          "short\_desc": "Complaint is being processed",  
          "updated\_at": "2023-01-15T10:04:01.812Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "xyz.sellerapp.com::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9960394039",  
              "email": "transactioncounterpartyapp@tcapp.com"  
            },  
            "person":  
            {  
              "name": "James Doe"  
            }  
          },  
          "cascaded\_level": 1  
        },  
        {  
          "respondent\_action": "CASCADED",  
          "short\_desc": "Complaint cascaded",  
          "updated\_at": "2023-01-15T10:05:00.267Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "xyz.sellerapp.com::nic2004:60232"  
            },  
            "contact":  
            {  
              "phone": "9960394039",  
              "email": "transactioncounterpartyapp@tcapp.com"  
            },  
            "person":  
            {  
              "name": "James Doe"  
            }  
          },  
          "cascaded\_level": 2  
        }  
        \]  
      },  
      "created\_at": "2023-01-15T10:00:00.469Z",  
      "updated\_at": "2023-01-15T10:05:00.267Z"  
    }  
  }  
}

### 4a. Logistics seller app responding marking the respondent action as processing signifying they have started processing the issue {#4a.-logistics-seller-app-responding-marking-the-respondent-action-as-processing-signifying-they-have-started-processing-the-issue}

/on\_issue  
{  
  "context":   
  {  
    "domain": "nic2004:60232",  
    "country": "IND",  
    "city": "std:080",  
    "action": "on\_issue",  
    "core\_version": "1.0.0",  
    "bap\_id": "xyz.sellerapp.com",  
    "bap\_uri": "https://xyz.sellerapp.counterpary.com",  
    "bpp\_id": "lmn.logisticssellerapp.com",  
    "bpp\_uri": "https://lmn.logisticssellerapp.counterpary.com",  
    "transaction\_id": "T2",  
    "message\_id": "M2",  
    "timestamp": "2023-01-15T10:15:15.932Z"  
  },  
  "message":  
  {[^42]  
    "issue":   
    {  
      "id": "I2",  
      "issue\_actions":  
      {  
        "respondent\_actions":  
        \[  
        {  
          "respondent\_action": "PROCESSING",  
          "short\_desc": "Complaint is being processed",  
          "updated\_at": "2023-01-15T10:04:01.812Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "xyz.sellerapp.com::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9960394039",  
              "email": "transactioncounterpartyapp@tcapp.com"  
            },  
            "person":  
            {  
              "name": "James Doe"  
            }  
          },  
          "cascaded\_level": 1  
        },  
        {  
          "respondent\_action": "CASCADED",  
          "short\_desc": "Complaint cascaded",  
          "updated\_at": "2023-01-15T10:05:00.267Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "xyz.sellerapp.com::nic2004:60232"  
            },  
            "contact":  
            {  
              "phone": "9960394039",  
              "email": "transactioncounterpartyapp@tcapp.com"  
            },  
            "person":  
            {  
              "name": "James Doe"  
            }  
          },  
        "cascaded\_level": 2  
        },  
        {  
          "respondent\_action": "PROCESSING",  
          "short\_desc": "Complaint is being processed",  
          "updated\_at": "2023-01-15T10:15:15.932Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "lmn.logisticssellerapp.counterpary.com::nic2004:60232"  
            },  
            "contact":  
            {  
              "phone": "9971394047",  
              "email": "cascadedcounterpartyapp@cascadapp.com"  
            },  
            "person":  
            {  
              "name": "Jimmy Doe"  
            }  
          },  
        "cascaded\_level": 2  
        }  
        \]  
      },  
      "created\_at": "2023-01-15T10:00:00.469Z",  
      "updated\_at": "2023-01-15T10:15:15.932Z"  
    }  
  }  
}

### 4b. Seller App providing the updated trail of actions to the buyer app {#4b.-seller-app-providing-the-updated-trail-of-actions-to-the-buyer-app}

/on\_issue\_status  
{  
  "context":   
  {  
    "domain": "ONDC:RET10",  
    "country": "IND",  
    "city": "std:080",  
    "action": "on\_issue\_status",  
    "core\_version": "1.0.0",  
    "bap\_id": "abc.interfacingapp.com",  
    "bap\_uri": "https://abc.buyerapp.com/ondc",  
    "bpp\_id": "xyz.sellerapp.com",  
    "bpp\_uri": "https://xyz.sellerapp.com/ondc",  
    "transaction\_id": "T1",  
    "message\_id": "M6",  
    "timestamp": "2023-01-15T10:15:15.932Z"  
  },  
  "message":  
  {  
    "issue":   
    {  
      "id": "I2",  
      "issue\_actions":  
      {  
        "respondent\_actions":  
        \[  
        {  
          "respondent\_action": "PROCESSING",  
          "short\_desc": "Complaint is being processed",  
          "updated\_at": "2023-01-15T10:04:01.812Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "xyz.sellerapp.com::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9960394039",  
              "email": "transactioncounterpartyapp@tcapp.com"  
            },  
            "person":  
            {  
              "name": "James Doe"  
            }  
          },  
          "cascaded\_level": 1  
        },  
        {  
          "respondent\_action": "CASCADED",  
          "short\_desc": "Complaint cascaded",  
          "updated\_at": "2023-01-15T10:05:00.267Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "xyz.sellerapp.com::nic2004:60232"  
            },  
            "contact":  
            {  
              "phone": "9960394039",  
              "email": "transactioncounterpartyapp@tcapp.com"  
            },  
            "person":  
            {  
              "name": "James Doe"  
            }  
          },  
        "cascaded\_level": 2  
        },  
        {  
          "respondent\_action": "PROCESSING",  
          "short\_desc": "Complaint is being processed",  
          "updated\_at": "2023-01-15T10:15:15.932Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "lmn.logisticssellerapp.counterpary.com::nic2004:60232"  
            },  
            "contact":  
            {  
              "phone": "9971394047",  
              "email": "cascadedcounterpartyapp@cascadapp.com"  
            },  
            "person":  
            {  
              "name": "Jimmy Doe"  
            }  
          },  
        "cascaded\_level": 2  
        }  
        \]  
      },  
      "created\_at": "2023-01-15T10:00:00.469Z",  
      "updated\_at": "2023-01-15T10:15:15.932Z"  
    }  
  }  
}

### 5\. Buyer app requesting status update on the complaint from the retail seller app {#5.-buyer-app-requesting-status-update-on-the-complaint-from-the-retail-seller-app}

/issue\_status  
{  
  "context":   
  {  
    "domain": "ONDC:RET10",  
    "country": "IND",  
    "city": "std:080",  
    "action": "issue\_status",  
    "core\_version": "1.0.0",  
    "bap\_id": "abc.interfacingapp.com",  
    "bap\_uri": "https://abc.buyerapp.com/ondc",  
    "bpp\_id": "xyz.sellerapp.com",  
    "bpp\_uri": "https://xyz.sellerapp.com/ondc",  
    "transaction\_id": "T1",  
    "message\_id": "M3",  
    "timestamp": "2023-01-15T10:30:00.867Z",  
    "ttl": "PT30S"  
  },  
  "message":  
  {  
    "issue\_id": "I2"  
  }  
}

### w6. Retail seller app requesting status update on the complaint from the logistics seller app {#w6.-retail-seller-app-requesting-status-update-on-the-complaint-from-the-logistics-seller-app}

/issue\_status  
{  
    "context":   
    {  
      "domain": "nic2004:60232",  
      "country": "IND",  
      "city": "std:080",  
      "action": "issue\_status",  
      "core\_version": "1.0.0",  
      "bap\_id": "xyz.sellerapp.com",  
      "bap\_uri": "https://xyz.sellerapp.counterpary.com/ondc",  
      "bpp\_id": "lmn.logisticssellerapp.com",  
      "bpp\_uri": "https://lmn.logisticssellerapp.counterpary.com/ondc",  
      "transaction\_id": "T2",  
      "message\_id": "M4",  
      "timestamp": "2023-01-15T10:30:02.267Z",  
      "ttl": "PT30S"  
    },  
    "message":  
    {  
      "issue\_id": "I2"  
    }  
}

### 7\. Logistics seller app providing status update on the complaint to the retail seller app {#7.-logistics-seller-app-providing-status-update-on-the-complaint-to-the-retail-seller-app}

/on\_issue\_status  
{  
  "context":  
  {  
    "domain": "nic2004:60232",  
    "country": "IND",  
    "city": "std:080",  
    "action": "on\_issue\_status",  
    "core\_version": "1.0.0",  
    "bap\_id": "xyz.sellerapp.com",  
    "bap\_uri": "https://xyz.sellerapp.counterpary.com/ondc",  
    "bpp\_id": "lmn.logisticssellerapp.com",  
    "bpp\_uri": "https://lmn.logisticssellerapp.counterpary.com/ondc",  
    "transaction\_id": "T2",  
    "message\_id": "M4",  
    "timestamp": "2023-01-15T10:30:15.935Z"  
  },  
  "message":  
  {[^43]  
    "issue":   
    {  
      "id": "I2",  
      "issue\_actions":  
      {  
        "respondent\_actions":  
        \[  
        {  
          "respondent\_action": "PROCESSING",  
          "short\_desc": "Complaint is being processed",  
          "updated\_at": "2023-01-15T10:04:01.812Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "xyz.sellerapp.com::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9960394039",  
              "email": "transactioncounterpartyapp@tcapp.com"  
            },  
            "person":  
            {  
              "name": "James Doe"  
            }  
          },  
          "cascaded\_level": 1  
        },  
        {  
          "respondent\_action": "CASCADED",  
          "short\_desc": "Complaint cascaded",  
          "updated\_at": "2023-01-15T10:05:00.267Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "xyz.sellerapp.com::nic2004:60232"  
            },  
            "contact":  
            {  
              "phone": "9960394039",  
              "email": "transactioncounterpartyapp@tcapp.com"  
            },  
            "person":  
            {  
              "name": "James Doe"  
            }  
          },  
          "cascaded\_level": 2  
        },  
        {  
          "respondent\_action": "PROCESSING",  
          "short\_desc": "Complaint is being processed",  
          "updated\_at": "2023-01-15T10:15:15.932Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "lmn.logisticssellerapp.counterpary.com::nic2004:60232"  
            },  
            "contact":  
            {  
              "phone": "9971394047",  
              "email": "cascadedcounterpartyapp@cascadapp.com"  
            },  
            "person":  
            {  
               "name": "Jimmy Doe"  
            }  
          },  
          "cascaded\_level": 2  
        },  
        {  
          "respondent\_action": "RESOLVED",  
          "short\_desc": "Complaint is being processed",  
          "updated\_at": "2023-01-15T10:30:15.935Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "lmn.logisticssellerapp.counterpary.com::nic2004:60232"  
            },  
            "contact":  
            {  
              "phone": "9971394047",  
              "email": "cascadedcounterpartyapp@cascadapp.com"  
            },  
            "person":  
            {  
              "name": "Jimmy Doe"  
            }  
          },  
          "cascaded\_level": 2  
        }  
        \]  
      },  
      "resolution\_provider":  
      {  
        "respondent\_info":  
        {  
          "type": "CASCADED-COUNTERPARTY-NP",  
          "organization":  
          {  
            "org":  
            {  
              "name": "lmn.logisticssellerapp.counterpary.com::nic2004:60232"  
            },  
            "contact":  
            {  
              "phone": "9971394047",  
              "email": "cascadedcounterpartyapp@cascadapp.com"  
            },  
            "person":  
            {  
              "name": "resolution provider org contact person name"  
            }  
          },  
          "resolution\_support":  
          {  
            "chat\_link": "https://cascadapp/chat-link",  
            "contact":  
            {  
              "phone": "9949595059",  
              "email": "respondantemail@resolutionprovider.com"  
            },  
            "gros":   
            \[  
            {  
              "person":   
              {  
                "name": "Fred J"  
              },  
              "contact":  
              {  
                "phone": "7865345298",  
                "email": "contact\_email@gro.com"  
              },  
              "gro\_type": "CASCADED-COUNTERPARTY-NP-GRO"  
            }  
            \]  
          }  
        }  
      },  
      "resolution":  
      {  
        "short\_desc": "issue resolution details",  
        "long\_desc": "remarks related to the resolution",  
        "action\_triggered": "REFUND",  
        "refund\_amount": "100"  
      },  
      "created\_at": "2023-01-15T10:00:00.469Z",  
      "updated\_at": "2023-01-15T10:30:15.935Z"  
    }  
  }  
}

### 8\. Retail Seller app providing status update on the complaint to the buyer app {#8.-retail-seller-app-providing-status-update-on-the-complaint-to-the-buyer-app}

#### Scenario 8a. Retail seller app cascading the resolution provided by the logistics provider to the buyer app {#scenario-8a.-retail-seller-app-cascading-the-resolution-provided-by-the-logistics-provider-to-the-buyer-app}

/on\_issue\_status  
{  
  "context":   
  {  
    "domain": "ONDC:RET10",  
    "country": "IND",  
    "city": "std:080",  
    "action": "on\_issue\_status",  
    "core\_version": "1.0.0",  
    "bap\_id": "abc.interfacingapp.com",  
    "bap\_uri": "https://abc.buyerapp.com/ondc",  
    "bpp\_id": "xyz.sellerapp.com",  
    "bpp\_uri": "https://xyz.sellerapp.com/ondc",  
    "transaction\_id": "T1",  
    "message\_id": "M3",  
    "timestamp": "2023-01-15T10:30:20.162Z"  
  },  
  "message":  
  {[^44]  
    "issue":   
    {  
      "id": "I2",  
      "issue\_actions":  
      {  
        "respondent\_actions":  
        \[  
        {  
          "respondent\_action": "PROCESSING",  
          "short\_desc": "Complaint is being processed",  
          "updated\_at": "2023-01-15T10:04:01.812Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "xyz.sellerapp.com::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9960394039",  
              "email": "transactioncounterpartyapp@tcapp.com"  
            },  
            "person":  
            {  
              "name": "James Doe"  
            }  
          },  
          "cascaded\_level": 1  
        },  
        {  
          "respondent\_action": "CASCADED",  
          "short\_desc": "Complaint cascaded",  
          "updated\_at": "2023-01-15T10:05:00.267Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "xyz.sellerapp.com::nic2004:60232"  
            },  
            "contact":  
            {  
              "phone": "9960394039",  
              "email": "transactioncounterpartyapp@tcapp.com"  
            },  
            "person":  
            {  
              "name": "James Doe"  
            }  
          },  
          "cascaded\_level": 2  
        },  
        {  
          "respondent\_action": "PROCESSING",  
          "short\_desc": "Complaint is being processed",  
          "updated\_at": "2023-01-15T10:15:15.932Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "lmn.logisticssellerapp.counterpary.com::nic2004:60232"  
            },  
            "contact":  
            {  
              "phone": "9971394047",  
              "email": "cascadedcounterpartyapp@cascadapp.com"  
            },  
            "person":  
            {  
              "name": "Jimmy Doe"  
            }  
          },  
          "cascaded\_level": 2  
        },  
        {  
          "respondent\_action": "RESOLVED",  
          "short\_desc": "Complaint is being resolved",  
          "updated\_at": "2023-01-15T10:30:15.932Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "lmn.logisticssellerapp.counterpary.com::nic2004:60232"  
            },  
            "contact":  
            {  
              "phone": "9971394047",  
              "email": "cascadedcounterpartyapp@cascadapp.com"  
            },  
            "person":  
            {  
              "name": "Jimmy Doe"  
            }  
          },  
          "cascaded\_level": 2  
        }  
        \]  
      },  
      "resolution\_provider":  
      {  
        "respondent\_info":  
        {  
          "type": "CASCADED-COUNTERPARTY-NP",  
          "organization":  
          {  
            "org":  
            {  
              "name": "lmn.logisticssellerapp.counterpary.com::nic2004:60232"  
            },  
            "contact":  
            {  
              "phone": "9971394047",  
              "email": "cascadedcounterpartyapp@cascadapp.com"  
            },  
            "person":  
            {  
              "name": "resolution provider org contact person name"  
            }  
          },  
          "resolution\_support":  
          {  
            "chat\_link": "https://cascadapp/chat-link",  
            "contact":  
            {  
              "phone": "9949595059",  
              "email": "respondantemail@resolutionprovider.com"  
            },  
            "gros":   
            \[  
            {  
              "person":   
              {  
                "name": "Fred J"  
              },  
              "contact":  
              {  
                "phone": "7865345298",  
                "email": "contact\_email@gro.com"  
              },  
              "gro\_type": "CASCADED-COUNTERPARTY-NP-GRO"  
            },  
            {  
              "person":   
              {  
                "name": "Sam D"  
              },  
              "contact":  
              {  
                "phone": "9605960796",  
                "email": "email@gro.com"  
              },  
              "gro\_type": "TRANSACTION-COUNTERPARTY-NP-GRO"  
            }  
            \]  
          }  
        }  
      },  
      "resolution":  
      {  
        "resolution": "issue resolution details",  
        "short\_desc": "remarks related to the resolution",  
        "action\_triggered": "REFUND",  
        "refund\_amount": "100"  
      },    {  
        "resolution": "issue resolution details",  
        "short\_desc": "remarks related to the resolution",  
        "a  
      "created\_at": "2023-01-15T10:00:00.469Z",  
      "updated\_at": "2023-01-15T10:30:15.935Z"  
    }  
  }  
}

#### Scenario 8b. Retail seller app providing the resolution provider to the buyer app {#scenario-8b.-retail-seller-app-providing-the-resolution-provider-to-the-buyer-app}

/on\_issue\_status  
{  
  "context":   
  {  
    "domain": "ONDC:RET10",  
    "country": "IND",  
    "city": "std:080",  
    "action": "on\_issue\_status",  
    "core\_version": "1.0.0",  
    "bap\_id": "abc.interfacingapp.com",  
    "bap\_uri": "https://abc.buyerapp.com/ondc",  
    "bpp\_id": "xyz.sellerapp.com",  
    "bpp\_uri": "https://xyz.sellerapp.com/ondc",  
    "transaction\_id": "T1",  
    "message\_id": "M3",  
    "timestamp": "2023-01-15T10:30:20.162Z"  
  },  
  "message":  
  {[^45]  
    "issue":   
    {  
      "id": "I2",  
      "issue\_actions":  
      {  
        "respondent\_actions":  
        \[  
        {  
          "respondent\_action": "PROCESSING",  
          "short\_desc": "Complaint is being processed",  
          "updated\_at": "2023-01-15T10:04:01.812Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "xyz.sellerapp.com::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9960394039",  
              "email": "transactioncounterpartyapp@tcapp.com"  
            },  
            "person":  
            {  
              "name": "James Doe"  
            }  
          },  
          "cascaded\_level": 1  
        },  
        {  
          "respondent\_action": "CASCADED",  
          "short\_desc": "Complaint cascaded",  
          "updated\_at": "2023-01-15T10:05:00.267Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "xyz.sellerapp.com::nic2004:60232"  
            },  
            "contact":  
            {  
              "phone": "9960394039",  
              "email": "transactioncounterpartyapp@tcapp.com"  
            },  
            "person":  
            {  
              "name": "James Doe"  
            }  
          },  
          "cascaded\_level": 2  
        },  
        {  
          "respondent\_action": "PROCESSING",  
          "short\_desc": "Complaint is being processed",  
          "updated\_at": "2023-01-15T10:15:15.932Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "lmn.logisticssellerapp.counterpary.com::nic2004:60232"  
            },  
            "contact":  
            {  
              "phone": "9971394047",  
              "email": "cascadedcounterpartyapp@cascadapp.com"  
            },  
            "person":  
            {  
              "name": "Jimmy Doe"  
            }  
          },  
          "cascaded\_level": 2  
        },  
        {  
          "respondent\_action": "RESOLVED",  
          "short\_desc": "Complaint is being resolved",  
          "updated\_at": "2023-01-15T10:30:15.932Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "lmn.logisticssellerapp.counterpary.com::nic2004:60232"  
            },  
            "contact":  
            {  
              "phone": "9971394047",  
              "email": "cascadedcounterpartyapp@cascadapp.com"  
            },  
            "person":  
            {  
              "name": "Jimmy Doe"  
            }  
          },  
          "cascaded\_level": 2  
        },  
        {  
          "respondent\_action": "RESOLVED",  
          "short\_desc": "Complaint is being resolved",  
          "updated\_at": "2023-01-15T10:30:20.162Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "xyz.sellerapp.com::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9960394039",  
              "email": "transactioncounterpartyapp@tcapp.com"  
            },  
            "person":  
            {  
              "name": "James Doe"  
            }  
          },  
          "cascaded\_level": 2  
        }  
        \]  
      },  
      "resolution\_provider":  
      {  
        "respondent\_info":  
        {  
          "type": "TRANSACTION-COUNTERPARTY-NP",  
          "organization":  
          {  
            "org":  
            {  
              "name": "xyz.sellerapp.com::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9059304940",  
              "email": "email@resolutionproviderorg.com"  
            },  
            "person":  
            {  
              "name": "resolution provider org contact person name"  
            }  
          },  
          "resolution\_support":  
          {  
            "chat\_link": "http://chat-link/respondent",  
            "contact":  
            {  
              "phone": "9949595059",  
              "email": "respondantemail@resolutionprovider.com"  
            },  
            "gros":   
            \[  
            {  
              "person":   
              {  
                "name": "Fred J"  
              },  
              "contact":  
              {  
                "phone": "7865345298",  
                "email": "contact\_email@gro.com"  
              },  
              "gro\_type": "CASCADED-COUNTERPARTY-NP-GRO"  
            },  
            {  
              "person":   
              {  
                "name": "Sam D"  
              },  
              "contact":  
              {  
                "phone": "9605960796",  
                "email": "email@gro.com"  
              },  
              "gro\_type": "TRANSACTION-COUNTERPARTY-NP-GRO"  
            }  
            \]  
          }  
        }  
      },  
      "resolution":  
      {  
        "short\_desc": "issue resolution details",  
        "long\_desc": "remarks related to the resolution",  
        "action\_triggered": "REFUND",  
        "refund\_amount": "150"  
      },  
      "created\_at": "2023-01-15T10:00:00.469Z",  
      "updated\_at": "2023-01-15T10:30:20.162Z"  
    }  
  }  
}

### 9a. Buyer app closing the complaint as closed by the buyer {#9a.-buyer-app-closing-the-complaint-as-closed-by-the-buyer}

/issue  
{  
  "context":   
  {  
    "domain": "ONDC:RET10",  
    "country": "IND",  
    "city": "std:080",  
    "action": "issue",  
    "core\_version": "1.0.0",  
    "bap\_id": "abc.interfacingapp.com",  
    "bap\_uri": "https://abc.buyerapp.com/ondc",  
    "bpp\_id": "xyz.sellerapp.com",  
    "bpp\_uri": "https://xyz.sellerapp.com/ondc",  
    "transaction\_id": "T1",  
    "message\_id": "M5",  
    "timestamp": "2023-01-15T10:41:30.723Z",  
    "ttl": "PT30S"  
  },  
  "message":  
  {[^46]  
    "issue":   
    {  
      "id": "I2",  
      "status": "CLOSED",  
      "issue\_actions":  
      {  
        "complainant\_actions":  
        \[  
        {  
          "complainant\_action": "OPEN",  
          "short\_desc": "Complaint created",  
          "updated\_at": "2023-01-15T10:00:00.469Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "abc.buyerapp.com/ondc::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9450394039",  
              "email": "interfacingapp@interface.com"  
            },  
            "person":  
            {  
              "name": "John Doe"  
            }  
          }  
        },  
        {  
          "complainant\_action": "CLOSE",  
          "short\_desc": "Complaint closed",  
          "updated\_at": "2023-01-15T10:41:30.723Z",  
          "updated\_by":   
          {  
            "org":  
            {  
             "name": "abc.buyerapp.com/ondc::ONDC:RET10"  
             },  
            "contact":  
            {  
              "phone": "9450394039",  
              "email": "interfacingapp@interface.com"  
            },  
            "person":  
            {  
              "name": "John Doe"  
            }  
          }  
        }  
        \]  
      },  
      "rating": "THUMBS-UP",  
      "created\_at": "2023-01-15T10:00:00.469Z",  
      "updated\_at": "2023-01-15T10:41:30.723Z"  
    }  
  }  
}

### 9b. Buyer app escalating the complaint as escalated by the buyer {#9b.-buyer-app-escalating-the-complaint-as-escalated-by-the-buyer}

/issue  
{  
  "context":   
  {  
    "domain": "ONDC:RET10",  
    "country": "IND",  
    "city": "std:080",  
    "action": "issue",  
    "core\_version": "1.0.0",  
    "bap\_id": "abc.interfacingapp.com",  
    "bap\_uri": "https://abc.buyerapp.com/ondc",  
    "bpp\_id": "xyz.sellerapp.com",  
    "bpp\_uri": "https://xyz.sellerapp.com/ondc",  
    "transaction\_id": "T1",  
    "message\_id": "M5",  
    "timestamp": "2023-01-15T10:41:30.723Z",  
    "ttl": "PT30S"  
  },  
  "message":  
  {  
    "issue":   
    {  
      "id": "I2",  
      "status": "OPEN",  
      "issue\_type": "GRIEVANCE",  
      "issue\_actions":  
      {  
        "complainant\_actions":  
        \[  
        {  
          "complainant\_action": "OPEN",  
          "short\_desc": "Complaint created",  
          "updated\_at": "2023-01-15T10:00:00.469Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "abc.buyerapp.com/ondc::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9450394039",  
              "email": "interfacingapp@interface.com"  
            },  
            "person":  
            {  
              "name": "John Doe"  
            }  
          }  
        },  
        {  
          "complainant\_action": "ESCALATE",  
          "short\_desc": "Not satisfied with the resolution provided",  
          "updated\_at": "2023-01-15T10:41:30.723Z",  
          "updated\_by":   
          {  
            "org":  
            {  
             "name": "abc.buyerapp.com/ondc::ONDC:RET10"  
             },  
            "contact":  
            {  
              "phone": "9450394039",  
              "email": "interfacingapp@interface.com"  
            },  
            "person":  
            {  
              "name": "John Doe"  
            }  
          }  
        }  
        \]  
      },  
      "created\_at": "2023-01-15T10:00:00.469Z",  
      "updated\_at": "2023-01-15T10:41:30.723Z"  
    }  
  }  
}

### 10a. Seller app closing the complaint to the logistics provider {#10a.-seller-app-closing-the-complaint-to-the-logistics-provider}

/issue  
{  
  "context":   
  {  
    "domain": "nic2004:60232",  
    "country": "IND",  
    "city": "std:080",  
    "action": "issue",  
    "core\_version": "1.0.0",  
    "bap\_id": "xyz.sellerapp.com",  
    "bap\_uri":≤\< "https://xyz.sellerapp.counterpary.com/ondc",  
    "bpp\_id": "lmn.logisticssellerapp.com",  
    "bpp\_uri": "https://lmn.logisticssellerapp.counterpary.com/ondc",  
    "transaction\_id": "T2",  
    "message\_id": "M7",  
    "timestamp": "2023-01-15T10:41:30.723Z",  
    "ttl": "PT30S"  
  },  
  "message":  
  {[^47]  
    "issue":   
    {  
      "id": "I2",  
      "status": "CLOSED",  
      "issue\_actions":  
      {  
        "complainant\_actions":  
        \[  
        {  
          "complainant\_action": "OPEN",  
          "short\_desc": "Complaint created",  
          "updated\_at": "2023-01-15T10:00:00.469Z",  
          "updated\_by":   
          {  
            "org":  
            {  
              "name": "abc.buyerapp.com/ondc::ONDC:RET10"  
            },  
            "contact":  
            {  
              "phone": "9450394039",  
              "email": "interfacingapp@interface.com"  
            },  
            "person":  
            {  
              "name": "John Doe"  
            }  
          }  
        },  
        {  
          "complainant\_action": "CLOSE",  
          "short\_desc": "Complaint closed",  
          "updated\_at": "2023-01-15T10:41:30.723Z",  
          "updated\_by":   
          {  
            "org":  
            {  
             "name": "abc.buyerapp.com/ondc::ONDC:RET10"  
             },  
            "contact":  
            {  
              "phone": "9450394039",  
              "email": "interfacingapp@interface.com"  
            },  
            "person":  
            {  
              "name": "John Doe"  
            }  
          }  
        }  
        \]  
      },  
      "rating": "THUMBS-UP",  
      "created\_at": "2023-01-15T10:00:00.469Z",  
      "updated\_at": "2023-01-15T10:41:30.723Z"  
    }  
  }  
}

## 

## 

## Error Codes

Please follow error codes as documented in Section 8 in the IGM Process flow document [here](https://docs.google.com/document/d/135OCfsi5jQ7wh4H_LOoMxb0T0ZrWDYy4LTBvpYS6k_w/edit?usp=sharing).

## FAQs {#faqs}

| Sr. | Question | Answer |
| :---: | ----- | ----- |
| 1 | What are the policies around Issue and Grievance Management? | ONDC Network Policies [Chapter 6](https://ondc-static-website-media.s3.ap-south-1.amazonaws.com/ondc-website-media/downloads/governance-and-policies/CHAPTER+%5B6%5D+Issue+and+Grievance+Management+Policy.pdf): IGM policies published on the ONDC website covers the policies around the issue and grievance management framework. |
| 2 | What is process for an NP for deploying IGM solution on their Production environment | Log verification followed by ops verification (QA testing) |
| 2 | What are the scenarios to be emulated for log verification? Where do NP submit logs? | [Test Case Scenarios for IGM Log Validation](https://drive.google.com/file/d/1O-11TsUGLLvYgq1Fo-HNIkGIlOelBJH4/view) |
| 3 | Who decides who will be the Respondent? Person or System? | The Interfacing app where the complaint is raised decides whom to traverse the complaint after their due assessment of the complaint. If the complainant is a buyer, the buyer app becomes the first respondent. If the buyer app thinks the retail-seller app is needed to resolve the complaint, they traverse the complaint and related information to that seller app. The seller app can in turn traverse the complaint to the logistics provider in case they have procured on-network logistics. In case the buyer app has procured on-network logistics, they have the choice to decide which app they want to traverse the complaint first. |
| 4 | What if the Seller app is not a network participant? Where will the seller raise the complaint? I mean in the TSP model. | In the case of the TSP model, the seller becomes the seller app. Any complaint that the seller (seller app) has with any other NP for a transaction executed over the network, they can use the IGM APIs. The issues between seller and TSP shall be off-network and off-IGM framework. |
| 5 | Is there a facility to seek clarification for an issue by NP1 to the Buyer/Seller ? | Yes, the issue\_status can be marked as 'need more info' and requisite information can be sought from the complainant |
| 6 | For tier 1 complaints, is the ticket marked resolved upon resolution by NPs or does it ask customers to accept or reject the resolution and then mark it complete? | Complaint is marked as 'resolved' by a respondent/ resolution provider. Complaint is marked as 'closed' by the complainant (or automatically marked if window expires). |
| 7 | Where the issue is between the buyer app and logistics app, is it mandatory to involve the seller app? Can the buyer app directly forward the complaint to the logistics app? | If the buyer app has procured on-network logistics, then yes, the buyer app can choose to traverse the complaint first to the logistics app then to the seller app, if required. In case the seller app has procured on-network logistics, then the buyer app has to traverse it to the seller app, who will then traverse it to the logistics app if required. |
| 8 | Wanted to know the scenarios where in seller can raise the complaint | Some scenarios include a. Issues with logistics provider app (in case of on network logistics) b. Reconciliation and Settlement related flows \- amount not settled |
| 9 | How will TAT breach by NPs be handled in the long term? | The handling of TAT is the responsibility of the respective NP. |
| 10 | While NPs use their respective Support systems, will ONDC maintain a repository of issues and grievances to monitor the SLA and closure. | ONDC is not going to maintain a repository of issues and grievances. However ONDC may ask for the same from the NP, as per the defined T\&C of the network policies. Aside from this, each NP has to appoint a certification agency to ensure the compliance to ONDC defined policies and procedures. The certification agency will perform the audit and validate the compliance to SLAs. |
| 11 | Will the IGM framework be only used for actual transaction level issues or inter NP issues such as settlement disputes be handled via the IGM framework | IGM framework will be used for handling complaints on the network identified through an id (transaction\_id, order\_id, fulfilment\_id, item\_id). This includes issue flows for the RSP framework as well. Tech issues will be handled through JIRA (currently being used). |
| 12 | whether Invoice and Tax related disputes among NPs such TDS certificate issuance, GST filings in 2A, Issue w.r.t. invoices, etc. will also be handled via IGM? | No, matters related to Tax, TDS certificates, GSTR 2A etc are to be handled outside IGM |
| 13 | Governance mechanism for technical issues between NPs | Technical issues will continue to be handled through JIRA |
| 14 | Are multiple complaints allowed within an order (i.e., for different items/ fulfilments) | Yes, multiple complaints are allowed with different network issue ids mapped with the specific object ids. Within an order a complaint can be raised for specific objects or specific fulfilments as required. The NPs have to maintain separate tickets for the same.  |
| 15 | Are multiple complaints allowed for a single item (item.id) simultaneously.  Order comprises of 1 item (1 item.id) in quantity of N (e.g., 5 pieces of the same shirt). What happens in case the buyer has two separate issues with the order (e.g., one shirt has a color mismatch and one shirt has broken buttons while the other three shirts are alright) Should the buyer app enable multiple ‘complaint tickets’ to be raised (and traversed) for the same item.id or will a single complaint be used to traverse information. So, if the buyer sequentially updates the complaints, does the complaint information/ description get updated or a new complaint. | The buyer app should enable raising two separate tickets for such scenarios and separate trails should be maintained by the NPs |
| 16 | A buyer raises a complaint with the buyer app. The buyer app traverses the complaint to the seller app with seller app procuring on-network logistics. Can seller app traverse only a certain part of the complaint as a new complaint ticket (a new network issue id) to the logistics service provider (lsp), as they may require assistance in only a specific part of the complaint | The seller app can choose to traverse only a certain part to the LSP with regards to getting resolution on a certain part of the complaint. But they should traverse with the same network issue id, in order to maintain the TATs as defined in the IGM policy where the time to provide a resolution will be dependent on the number of cascaded NPs involved. Creating a new network issue id ticket for sub parts with its own timelines will create conflicts in complaint handling timelines. |
| 17 |  |  |

[^1]:   Refer domain specific IGM category/ sub category sheets

[^2]:  [FAQs](https://docs.google.com/document/d/135OCfsi5jQ7wh4H_LOoMxb0T0ZrWDYy4LTBvpYS6k_w/edit#heading=h.r35gw9smm4dp) for reference

[^3]:  The example illustrates an item category complaint. Category being leveraged in a transaction depends on the domain of the transaction

[^4]:  Context object changes as per the respective domain transaction API specifications, including the version attribute that changes its value based on the domain version e.g. Retail \- 1.2.0 or TRV10 2.0.1

[^5]:  Domain code specific to the transaction executed

[^6]:  Should be the same as in core transaction APIs

[^7]:  Enums are as per the transaction domain, refer to the specific category sheet

[^8]:  Enums defined [here](https://docs.google.com/spreadsheets/d/1Ca9I6smiLKPgXT-dYHBWineV-6P_La9r/edit?usp=sharing&ouid=100595989766867836454&rtpof=true&sd=true)

[^9]:  Optional key

[^10]:  Optional key, enums depend on domain of the transaction

[^11]:  Mandatory in case the issue\_category is ITEM, optional in case the issue\_category is FULFILLMENT, should contain items for which the complaint has been raised

[^12]:  Conditionally mandatory if the issue is raised for an item

[^13]:  Optional in case the issue\_category is ITEM, mandatory in case the issue\_category is FULFILLMENT

[^14]:  Optional key

[^15]:  Optional key, may include trail of complaint from sending NPs’ CRM/ Support system

[^16]:  Conditional mandatory for certain issue categories \- sub categories \- Item \[ITM02, ITM03, ITM04, ITM05\] ; Fulfillment \[FLM04\]

[^17]:  Enums are CONSUMER, SELLER, INTERFACING NP

[^18]:  Provided as per the maximum time for complaint resolution provided in the [IGM Policy](https://ondc-static-website-media.s3.ap-south-1.amazonaws.com/ondc-website-media/downloads/governance-and-policies/CHAPTER+%5B6%5D+Issue+and+Grievance+Management+Policy.pdf), buyer apps may provide reduced times based on the domain or the complaint requirement

[^19]:  Enums are OPEN, CLOSED; This is maintained by the interfacing app

[^20]:  Enums are ISSUE, GRIEVANCE, DISPUTE

[^21]:  Enums are OPEN, ESCALATE, CLOSE

[^22]:  Information about the buyer app (interfacing app) and respective representative

[^23]:  "org" should be NP’s "subscriber\_id::domain"

[^24]:  Enums are PROCESSING, CASCADED, RESOLVED, NEED-MORE-INFO

[^25]:  Information about the seller app (interfacing app) and respective representative

[^26]:  Details of a representative of the contact/ support team of the seller app/ respondent NP

[^27]:  Optional key, the default value will be 1, whenever the issue will be cascaded to the next level the value of this field should be incremented by 1

[^28]:  When the latest respondent\_action is "RESOLVED", the resolution\_provider object is mandatory

[^29]:  Enums are INTERFACING-NP, TRANSACTION-COUNTERPARTY-NP, CASCADED-COUNTERPARTY-NP

[^30]:  Optional key, provided by the resolution provider in case they want to provide a communication link for the buyer to interact with the seller

[^31]:  Resolution provider shares their GRO information as part of the resolution support in L1 (issue) flow

[^32]: Enums are  INTERFACING-NP-GRO, TRANSACTION-COUNTERPARTY-NP-GRO, CASCADED-COUNTERPARTY-NP-GRO

[^33]:  When the latest respondent\_action is "RESOLVED", the resolution\_provider object is mandatory

[^34]:  Optional key to provide additional information around the resolution provided

[^35]:  Enums are REFUND, REPLACEMENT, CANCEL, NO-ACTION, RECONCILED, NOT-RECONCILED

[^36]:  Mandatory only when the action\_triggered is REFUND

[^37]:  Optional attribute, depending on whether the buyer (complainant has provided any rating or not)

[^38]:  Optional flow, not covered as part of the MVP flow

[^39]:  Refer to Scenario 1 section 1 footnotes for keys that are optional

[^40]:  Optional key

[^41]:  When the complaint is cascaded, the respondent sends the complaint\_actions trail as captured so far and the respondent\_actions capturing their trail of actions

[^42]:  Refer to Scenario 1 section 2 footnotes for keys that are optional

[^43]:  Refer to Scenario 1 section 4 footnotes for keys that are optional

[^44]:  Refer to Scenario 1 section 4 footnotes for keys that are optional

[^45]:  Refer to Scenario 1 section 4 footnotes for keys that are optional

[^46]:  Refer to Scenario 1 section 5a footnotes for keys that are optional

[^47]:  Refer to Scenario 1 section 5a footnotes for keys that are optional