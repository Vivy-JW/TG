using System;
using SpecDesignerCommon;

namespace SpecDesigner.Infrastructure.Event
{
	// Token: 0x02000005 RID: 5
	public class CreateFunctionContentArgs
	{
		// Token: 0x17000001 RID: 1
		// (get) Token: 0x0600000A RID: 10 RVA: 0x000021AD File Offset: 0x000003AD
		// (set) Token: 0x0600000B RID: 11 RVA: 0x000021B5 File Offset: 0x000003B5
		public PackageKey ProgramKey { get; set; }

		// Token: 0x17000002 RID: 2
		// (get) Token: 0x0600000C RID: 12 RVA: 0x000021BE File Offset: 0x000003BE
		// (set) Token: 0x0600000D RID: 13 RVA: 0x000021C6 File Offset: 0x000003C6
		public string Content { get; set; }
	}
}
