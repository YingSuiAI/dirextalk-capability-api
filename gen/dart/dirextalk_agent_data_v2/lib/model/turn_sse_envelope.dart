//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of dirextalk_agent_data_v2;

class TurnSseEnvelope {
  /// Returns a new [TurnSseEnvelope] instance.
  TurnSseEnvelope({
    required this.operationId,
    required this.turnId,
    required this.conversationId,
    required this.sequence,
    required this.type,
    required this.payload,
    required this.createdAt,
  });

  /// Durable operation UUID.
  String operationId;

  /// Durable Turn UUID.
  String turnId;

  /// Durable conversation UUID.
  String conversationId;

  /// Monotonic per-operation replay cursor.
  ///
  /// Minimum value: 0
  int sequence;

  String type;

  /// Durable Turn event projection or safe ErrorEnvelope.
  Object? payload;

  DateTime createdAt;

  @override
  bool operator ==(Object other) => identical(this, other) || other is TurnSseEnvelope &&
    other.operationId == operationId &&
    other.turnId == turnId &&
    other.conversationId == conversationId &&
    other.sequence == sequence &&
    other.type == type &&
    other.payload == payload &&
    other.createdAt == createdAt;

  @override
  int get hashCode =>
    // ignore: unnecessary_parenthesis
    (operationId.hashCode) +
    (turnId.hashCode) +
    (conversationId.hashCode) +
    (sequence.hashCode) +
    (type.hashCode) +
    (payload == null ? 0 : payload!.hashCode) +
    (createdAt.hashCode);

  @override
  String toString() => 'TurnSseEnvelope[operationId=$operationId, turnId=$turnId, conversationId=$conversationId, sequence=$sequence, type=$type, payload=$payload, createdAt=$createdAt]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
      json[r'operation_id'] = this.operationId;
      json[r'turn_id'] = this.turnId;
      json[r'conversation_id'] = this.conversationId;
      json[r'sequence'] = this.sequence;
      json[r'type'] = this.type;
    if (this.payload != null) {
      json[r'payload'] = this.payload;
    } else {
      json[r'payload'] = null;
    }
      json[r'created_at'] = this.createdAt.toUtc().toIso8601String();
    return json;
  }

  /// Returns a new [TurnSseEnvelope] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static TurnSseEnvelope? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        requiredKeys.forEach((key) {
          assert(json.containsKey(key), 'Required key "TurnSseEnvelope[$key]" is missing from JSON.');
          assert(json[key] != null, 'Required key "TurnSseEnvelope[$key]" has a null value in JSON.');
        });
        return true;
      }());

      return TurnSseEnvelope(
        operationId: mapValueOfType<String>(json, r'operation_id')!,
        turnId: mapValueOfType<String>(json, r'turn_id')!,
        conversationId: mapValueOfType<String>(json, r'conversation_id')!,
        sequence: mapValueOfType<int>(json, r'sequence')!,
        type: mapValueOfType<String>(json, r'type')!,
        payload: mapValueOfType<Object>(json, r'payload'),
        createdAt: mapDateTime(json, r'created_at', r'')!,
      );
    }
    return null;
  }

  static List<TurnSseEnvelope> listFromJson(dynamic json, {bool growable = false,}) {
    final result = <TurnSseEnvelope>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = TurnSseEnvelope.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, TurnSseEnvelope> mapFromJson(dynamic json) {
    final map = <String, TurnSseEnvelope>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = TurnSseEnvelope.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of TurnSseEnvelope-objects as value to a dart map
  static Map<String, List<TurnSseEnvelope>> mapListFromJson(dynamic json, {bool growable = false,}) {
    final map = <String, List<TurnSseEnvelope>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = TurnSseEnvelope.listFromJson(entry.value, growable: growable,);
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'operation_id',
    'turn_id',
    'conversation_id',
    'sequence',
    'type',
    'payload',
    'created_at',
  };
}

