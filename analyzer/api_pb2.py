# -*- coding: utf-8 -*-
# Generated by the protocol buffer compiler.  DO NOT EDIT!
# source: api.proto
"""Generated protocol buffer code."""
from google.protobuf.internal import builder as _builder
from google.protobuf import descriptor as _descriptor
from google.protobuf import descriptor_pool as _descriptor_pool
from google.protobuf import symbol_database as _symbol_database
# @@protoc_insertion_point(imports)

_sym_db = _symbol_database.Default()




DESCRIPTOR = _descriptor_pool.Default().AddSerializedFile(b'\n\tapi.proto\x12\x03\x61pi\"\x07\n\x05\x45mpty\">\n\x08\x41udioBuf\x12\x11\n\tsensor_id\x18\x01 \x01(\x05\x12\x12\n\nseq_offset\x18\x02 \x01(\r\x12\x0b\n\x03pcm\x18\x03 \x01(\x0c\"2\n\rStatusRequest\x12\x11\n\tsensor_id\x18\x01 \x01(\x05\x12\x0e\n\x06status\x18\x02 \x01(\t\"!\n\x0eStatusResponse\x12\x0f\n\x07success\x18\x01 \x01(\x08\x32\x39\n\x0f\x41nalyzerService\x12&\n\x07\x41nalyze\x12\r.api.AudioBuf\x1a\n.api.Empty(\x01\x32I\n\x0eGatewayService\x12\x37\n\x0cUpdateStatus\x12\x12.api.StatusRequest\x1a\x13.api.StatusResponseB\tZ\x07./apipbb\x06proto3')

_builder.BuildMessageAndEnumDescriptors(DESCRIPTOR, globals())
_builder.BuildTopDescriptorsAndMessages(DESCRIPTOR, 'api_pb2', globals())
if _descriptor._USE_C_DESCRIPTORS == False:

  DESCRIPTOR._options = None
  DESCRIPTOR._serialized_options = b'Z\007./apipb'
  _EMPTY._serialized_start=18
  _EMPTY._serialized_end=25
  _AUDIOBUF._serialized_start=27
  _AUDIOBUF._serialized_end=89
  _STATUSREQUEST._serialized_start=91
  _STATUSREQUEST._serialized_end=141
  _STATUSRESPONSE._serialized_start=143
  _STATUSRESPONSE._serialized_end=176
  _ANALYZERSERVICE._serialized_start=178
  _ANALYZERSERVICE._serialized_end=235
  _GATEWAYSERVICE._serialized_start=237
  _GATEWAYSERVICE._serialized_end=310
# @@protoc_insertion_point(module_scope)
