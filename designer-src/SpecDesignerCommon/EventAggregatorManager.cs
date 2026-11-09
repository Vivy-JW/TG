using System;
using System.Collections.Generic;
using Microsoft.Practices.Prism.Events;

namespace SpecDesignerCommon
{
	// Token: 0x020000F0 RID: 240
	public class EventAggregatorManager
	{
		// Token: 0x060007FF RID: 2047 RVA: 0x000237C0 File Offset: 0x000219C0
		public static EventAggregator Get(PackageKey key)
		{
			if (EventAggregatorManager._eventAggregatorMap.ContainsKey(key))
			{
				return EventAggregatorManager._eventAggregatorMap[key];
			}
			throw new Exception(string.Format("{0} EventAggregator not exist", key));
		}

		// Token: 0x17000232 RID: 562
		// (get) Token: 0x06000800 RID: 2048 RVA: 0x000237EB File Offset: 0x000219EB
		public static EventAggregator Global
		{
			get
			{
				if (EventAggregatorManager.eventAggregator == null)
				{
					EventAggregatorManager.eventAggregator = new EventAggregator();
				}
				return EventAggregatorManager.eventAggregator;
			}
		}

		// Token: 0x06000801 RID: 2049 RVA: 0x00023803 File Offset: 0x00021A03
		public static void CreateInstance(PackageKey key)
		{
			if (EventAggregatorManager._eventAggregatorMap.ContainsKey(key))
			{
				throw new Exception(string.Format("{0} EventAggregator exist", key));
			}
			EventAggregatorManager._eventAggregatorMap.Add(key, new EventAggregator());
		}

		// Token: 0x06000802 RID: 2050 RVA: 0x00023833 File Offset: 0x00021A33
		public static void Remove(PackageKey key)
		{
			EventAggregatorManager._eventAggregatorMap.Remove(key);
		}

		// Token: 0x06000803 RID: 2051 RVA: 0x00023841 File Offset: 0x00021A41
		public static bool ContainsKey(PackageKey key)
		{
			return EventAggregatorManager._eventAggregatorMap.ContainsKey(key);
		}

		// Token: 0x040002DA RID: 730
		private static Dictionary<PackageKey, EventAggregator> _eventAggregatorMap = new Dictionary<PackageKey, EventAggregator>(new PackageKey.PackageKeyComparer());

		// Token: 0x040002DB RID: 731
		public static EventAggregator eventAggregator = null;
	}
}
