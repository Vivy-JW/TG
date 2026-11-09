using System;

namespace SpecDesigner.Infrastructure.Event
{
	// Token: 0x0200000B RID: 11
	public class RemoveFunctionInformation
	{
		// Token: 0x17000007 RID: 7
		// (get) Token: 0x0600001B RID: 27 RVA: 0x0000223B File Offset: 0x0000043B
		// (set) Token: 0x0600001C RID: 28 RVA: 0x00002243 File Offset: 0x00000443
		public Guid ID { get; private set; }

		// Token: 0x0600001D RID: 29 RVA: 0x0000224C File Offset: 0x0000044C
		public RemoveFunctionInformation(Guid id)
		{
			this.ID = id;
		}
	}
}
