//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of dirextalk_agent_data_v2;

class ErrorEnvelope {
  /// Returns a new [ErrorEnvelope] instance.
  ErrorEnvelope({
    required this.code,
    required this.message,
    required this.category,
    required this.retryable,
    this.retryAfterMs,
    required this.requestId,
    this.operationId,
    this.turnId,
    this.details,
  });

  String code;

  /// Safe client-facing message with no secret or raw upstream data.
  String message;

  ErrorCategory category;

  bool retryable;

  /// Minimum value: 0
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  int? retryAfterMs;

  /// Safe request correlation identifier.
  String requestId;

  /// Durable operation UUID.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? operationId;

  /// Durable Turn UUID.
  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  String? turnId;

  ///
  /// Please note: This property should have been non-nullable! Since the specification file
  /// does not include a default value (using the "default:" property), however, the generated
  /// source code must fall back to having a nullable type.
  /// Consider adding a "default:" property in the specification file to hide this note.
  ///
  ErrorDetails? details;

  @override
  bool operator ==(Object other) => identical(this, other) || other is ErrorEnvelope &&
    other.code == code &&
    other.message == message &&
    other.category == category &&
    other.retryable == retryable &&
    other.retryAfterMs == retryAfterMs &&
    other.requestId == requestId &&
    other.operationId == operationId &&
    other.turnId == turnId &&
    other.details == details;

  @override
  int get hashCode =>
    // ignore: unnecessary_parenthesis
    (code.hashCode) +
    (message.hashCode) +
    (category.hashCode) +
    (retryable.hashCode) +
    (retryAfterMs == null ? 0 : retryAfterMs!.hashCode) +
    (requestId.hashCode) +
    (operationId == null ? 0 : operationId!.hashCode) +
    (turnId == null ? 0 : turnId!.hashCode) +
    (details == null ? 0 : details!.hashCode);

  @override
  String toString() => 'ErrorEnvelope[code=$code, message=$message, category=$category, retryable=$retryable, retryAfterMs=$retryAfterMs, requestId=$requestId, operationId=$operationId, turnId=$turnId, details=$details]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
      json[r'code'] = this.code;
      json[r'message'] = this.message;
      json[r'category'] = this.category;
      json[r'retryable'] = this.retryable;
    if (this.retryAfterMs != null) {
      json[r'retry_after_ms'] = this.retryAfterMs;
    } else {
      json[r'retry_after_ms'] = null;
    }
      json[r'request_id'] = this.requestId;
    if (this.operationId != null) {
      json[r'operation_id'] = this.operationId;
    } else {
      json[r'operation_id'] = null;
    }
    if (this.turnId != null) {
      json[r'turn_id'] = this.turnId;
    } else {
      json[r'turn_id'] = null;
    }
    if (this.details != null) {
      json[r'details'] = this.details;
    } else {
      json[r'details'] = null;
    }
    return json;
  }

  /// Returns a new [ErrorEnvelope] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static ErrorEnvelope? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        requiredKeys.forEach((key) {
          assert(json.containsKey(key), 'Required key "ErrorEnvelope[$key]" is missing from JSON.');
          assert(json[key] != null, 'Required key "ErrorEnvelope[$key]" has a null value in JSON.');
        });
        return true;
      }());

      return ErrorEnvelope(
        code: mapValueOfType<String>(json, r'code')!,
        message: mapValueOfType<String>(json, r'message')!,
        category: ErrorCategory.fromJson(json[r'category'])!,
        retryable: mapValueOfType<bool>(json, r'retryable')!,
        retryAfterMs: mapValueOfType<int>(json, r'retry_after_ms'),
        requestId: mapValueOfType<String>(json, r'request_id')!,
        operationId: mapValueOfType<String>(json, r'operation_id'),
        turnId: mapValueOfType<String>(json, r'turn_id'),
        details: ErrorDetails.fromJson(json[r'details']),
      );
    }
    return null;
  }

  static List<ErrorEnvelope> listFromJson(dynamic json, {bool growable = false,}) {
    final result = <ErrorEnvelope>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = ErrorEnvelope.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, ErrorEnvelope> mapFromJson(dynamic json) {
    final map = <String, ErrorEnvelope>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = ErrorEnvelope.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of ErrorEnvelope-objects as value to a dart map
  static Map<String, List<ErrorEnvelope>> mapListFromJson(dynamic json, {bool growable = false,}) {
    final map = <String, List<ErrorEnvelope>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = ErrorEnvelope.listFromJson(entry.value, growable: growable,);
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'code',
    'message',
    'category',
    'retryable',
    'request_id',
  };
}

