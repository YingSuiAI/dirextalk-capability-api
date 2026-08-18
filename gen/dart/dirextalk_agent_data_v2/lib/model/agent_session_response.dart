//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//
// @dart=2.18

// ignore_for_file: unused_element, unused_import
// ignore_for_file: always_put_required_named_parameters_first
// ignore_for_file: constant_identifier_names
// ignore_for_file: lines_longer_than_80_chars

part of dirextalk_agent_data_v2;

class AgentSessionResponse {
  /// Returns a new [AgentSessionResponse] instance.
  AgentSessionResponse({
    required this.ticket,
    required this.expiresAt,
    required this.serverTime,
    required this.basePath,
    required this.sessionId,
    this.scopes = const {},
  });

  /// Short-lived compact EdDSA JWS bearer ticket.
  String ticket;

  /// Ticket expiration as RFC 3339 UTC.
  DateTime expiresAt;

  /// Ticket issuance time as RFC 3339 UTC.
  DateTime serverTime;

  AgentSessionResponseBasePathEnum basePath;

  /// Stable client-selected session UUID reused during refresh.
  String sessionId;

  Set<AgentDataScope> scopes;

  @override
  bool operator ==(Object other) => identical(this, other) || other is AgentSessionResponse &&
    other.ticket == ticket &&
    other.expiresAt == expiresAt &&
    other.serverTime == serverTime &&
    other.basePath == basePath &&
    other.sessionId == sessionId &&
    _deepEquality.equals(other.scopes, scopes);

  @override
  int get hashCode =>
    // ignore: unnecessary_parenthesis
    (ticket.hashCode) +
    (expiresAt.hashCode) +
    (serverTime.hashCode) +
    (basePath.hashCode) +
    (sessionId.hashCode) +
    (scopes.hashCode);

  @override
  String toString() => 'AgentSessionResponse[ticket=$ticket, expiresAt=$expiresAt, serverTime=$serverTime, basePath=$basePath, sessionId=$sessionId, scopes=$scopes]';

  Map<String, dynamic> toJson() {
    final json = <String, dynamic>{};
      json[r'ticket'] = this.ticket;
      json[r'expires_at'] = this.expiresAt.toUtc().toIso8601String();
      json[r'server_time'] = this.serverTime.toUtc().toIso8601String();
      json[r'base_path'] = this.basePath;
      json[r'session_id'] = this.sessionId;
      json[r'scopes'] = this.scopes.toList(growable: false);
    return json;
  }

  /// Returns a new [AgentSessionResponse] instance and imports its values from
  /// [value] if it's a [Map], null otherwise.
  // ignore: prefer_constructors_over_static_methods
  static AgentSessionResponse? fromJson(dynamic value) {
    if (value is Map) {
      final json = value.cast<String, dynamic>();

      // Ensure that the map contains the required keys.
      // Note 1: the values aren't checked for validity beyond being non-null.
      // Note 2: this code is stripped in release mode!
      assert(() {
        requiredKeys.forEach((key) {
          assert(json.containsKey(key), 'Required key "AgentSessionResponse[$key]" is missing from JSON.');
          assert(json[key] != null, 'Required key "AgentSessionResponse[$key]" has a null value in JSON.');
        });
        return true;
      }());

      return AgentSessionResponse(
        ticket: mapValueOfType<String>(json, r'ticket')!,
        expiresAt: mapDateTime(json, r'expires_at', r'')!,
        serverTime: mapDateTime(json, r'server_time', r'')!,
        basePath: AgentSessionResponseBasePathEnum.fromJson(json[r'base_path'])!,
        sessionId: mapValueOfType<String>(json, r'session_id')!,
        scopes: AgentDataScope.listFromJson(json[r'scopes']).toSet(),
      );
    }
    return null;
  }

  static List<AgentSessionResponse> listFromJson(dynamic json, {bool growable = false,}) {
    final result = <AgentSessionResponse>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AgentSessionResponse.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }

  static Map<String, AgentSessionResponse> mapFromJson(dynamic json) {
    final map = <String, AgentSessionResponse>{};
    if (json is Map && json.isNotEmpty) {
      json = json.cast<String, dynamic>(); // ignore: parameter_assignments
      for (final entry in json.entries) {
        final value = AgentSessionResponse.fromJson(entry.value);
        if (value != null) {
          map[entry.key] = value;
        }
      }
    }
    return map;
  }

  // maps a json object with a list of AgentSessionResponse-objects as value to a dart map
  static Map<String, List<AgentSessionResponse>> mapListFromJson(dynamic json, {bool growable = false,}) {
    final map = <String, List<AgentSessionResponse>>{};
    if (json is Map && json.isNotEmpty) {
      // ignore: parameter_assignments
      json = json.cast<String, dynamic>();
      for (final entry in json.entries) {
        map[entry.key] = AgentSessionResponse.listFromJson(entry.value, growable: growable,);
      }
    }
    return map;
  }

  /// The list of required keys that must be present in a JSON.
  static const requiredKeys = <String>{
    'ticket',
    'expires_at',
    'server_time',
    'base_path',
    'session_id',
    'scopes',
  };
}


class AgentSessionResponseBasePathEnum {
  /// Instantiate a new enum with the provided [value].
  const AgentSessionResponseBasePathEnum._(this.value);

  /// The underlying value of this enum member.
  final String value;

  @override
  String toString() => value;

  String toJson() => value;

  static const slashAgentSlashV1 = AgentSessionResponseBasePathEnum._(r'/agent/v1');

  /// List of all possible values in this [enum][AgentSessionResponseBasePathEnum].
  static const values = <AgentSessionResponseBasePathEnum>[
    slashAgentSlashV1,
  ];

  static AgentSessionResponseBasePathEnum? fromJson(dynamic value) => AgentSessionResponseBasePathEnumTypeTransformer().decode(value);

  static List<AgentSessionResponseBasePathEnum> listFromJson(dynamic json, {bool growable = false,}) {
    final result = <AgentSessionResponseBasePathEnum>[];
    if (json is List && json.isNotEmpty) {
      for (final row in json) {
        final value = AgentSessionResponseBasePathEnum.fromJson(row);
        if (value != null) {
          result.add(value);
        }
      }
    }
    return result.toList(growable: growable);
  }
}

/// Transformation class that can [encode] an instance of [AgentSessionResponseBasePathEnum] to String,
/// and [decode] dynamic data back to [AgentSessionResponseBasePathEnum].
class AgentSessionResponseBasePathEnumTypeTransformer {
  factory AgentSessionResponseBasePathEnumTypeTransformer() => _instance ??= const AgentSessionResponseBasePathEnumTypeTransformer._();

  const AgentSessionResponseBasePathEnumTypeTransformer._();

  String encode(AgentSessionResponseBasePathEnum data) => data.value;

  /// Decodes a [dynamic value][data] to a AgentSessionResponseBasePathEnum.
  ///
  /// If [allowNull] is true and the [dynamic value][data] cannot be decoded successfully,
  /// then null is returned. However, if [allowNull] is false and the [dynamic value][data]
  /// cannot be decoded successfully, then an [UnimplementedError] is thrown.
  ///
  /// The [allowNull] is very handy when an API changes and a new enum value is added or removed,
  /// and users are still using an old app with the old code.
  AgentSessionResponseBasePathEnum? decode(dynamic data, {bool allowNull = true}) {
    if (data != null) {
      switch (data) {
        case r'/agent/v1': return AgentSessionResponseBasePathEnum.slashAgentSlashV1;
        default:
          if (!allowNull) {
            throw ArgumentError('Unknown enum value to decode: $data');
          }
      }
    }
    return null;
  }

  /// Singleton [AgentSessionResponseBasePathEnumTypeTransformer] instance.
  static AgentSessionResponseBasePathEnumTypeTransformer? _instance;
}


