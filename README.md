# Protocol Documentation
<a name="top"/>

## Table of Contents

- [played.proto](#played.proto)
    - [AddUserRequest](#played.AddUserRequest)
    - [AddUserResponse](#played.AddUserResponse)
    - [CheckWhiteListResponse](#played.CheckWhiteListResponse)
    - [CheckWhitelistRequest](#played.CheckWhitelistRequest)
    - [GameEntry](#played.GameEntry)
    - [GetPlayedRequest](#played.GetPlayedRequest)
    - [GetPlayedResponse](#played.GetPlayedResponse)
    - [RemoveUserRequest](#played.RemoveUserRequest)
    - [RemoveUserResponse](#played.RemoveUserResponse)
    - [SendPlayedRequest](#played.SendPlayedRequest)
    - [SendPlayedResponse](#played.SendPlayedResponse)
  
  
  
    - [Played](#played.Played)
  

- [Scalar Value Types](#scalar-value-types)



<a name="played.proto"/>
<p align="right"><a href="#top">Top</a></p>

## played.proto



<a name="played.AddUserRequest"/>

### AddUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [string](#string) |  |  |






<a name="played.AddUserResponse"/>

### AddUserResponse







<a name="played.CheckWhiteListResponse"/>

### CheckWhiteListResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| whitelisted | [bool](#bool) |  |  |






<a name="played.CheckWhitelistRequest"/>

### CheckWhitelistRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [string](#string) |  |  |






<a name="played.GameEntry"/>

### GameEntry



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| name | [string](#string) |  |  |
| dur | [int32](#int32) |  |  |






<a name="played.GetPlayedRequest"/>

### GetPlayedRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [string](#string) |  |  |






<a name="played.GetPlayedResponse"/>

### GetPlayedResponse



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| games | [GameEntry](#played.GameEntry) | repeated |  |
| first | [string](#string) |  |  |
| last | [string](#string) |  |  |






<a name="played.RemoveUserRequest"/>

### RemoveUserRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [string](#string) |  |  |






<a name="played.RemoveUserResponse"/>

### RemoveUserResponse







<a name="played.SendPlayedRequest"/>

### SendPlayedRequest



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| user | [string](#string) |  |  |
| game | [string](#string) |  |  |






<a name="played.SendPlayedResponse"/>

### SendPlayedResponse






 

 

 


<a name="played.Played"/>

### Played


| Method Name | Request Type | Response Type | Description |
| ----------- | ------------ | ------------- | ------------|
| SendPlayed | [SendPlayedRequest](#played.SendPlayedRequest) | [SendPlayedResponse](#played.SendPlayedRequest) |  |
| GetPlayed | [GetPlayedRequest](#played.GetPlayedRequest) | [GetPlayedResponse](#played.GetPlayedRequest) |  |
| AddUser | [AddUserRequest](#played.AddUserRequest) | [AddUserResponse](#played.AddUserRequest) |  |
| RemoveUser | [RemoveUserRequest](#played.RemoveUserRequest) | [RemoveUserResponse](#played.RemoveUserRequest) |  |
| CheckWhitelist | [CheckWhitelistRequest](#played.CheckWhitelistRequest) | [CheckWhiteListResponse](#played.CheckWhitelistRequest) |  |

 



## Scalar Value Types

| .proto Type | Notes | C++ Type | Java Type | Python Type |
| ----------- | ----- | -------- | --------- | ----------- |
| <a name="double" /> double |  | double | double | float |
| <a name="float" /> float |  | float | float | float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long |
| <a name="bool" /> bool |  | bool | boolean | boolean |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str |

