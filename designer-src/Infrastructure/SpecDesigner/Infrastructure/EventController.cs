using System;
using Microsoft.Practices.Prism.Events;

namespace SpecDesigner.Infrastructure
{
	// Token: 0x0200001B RID: 27
	public class EventController
	{
		// Token: 0x0600004B RID: 75 RVA: 0x000023E0 File Offset: 0x000005E0
		public static EventAggregator GetInstance()
		{
			return EventController._eventAggregator;
		}

		// Token: 0x0600004C RID: 76 RVA: 0x000023E7 File Offset: 0x000005E7
		private EventController()
		{
		}

		// Token: 0x0400001C RID: 28
		private static EventAggregator _eventAggregator = new EventAggregator();
	}
}
