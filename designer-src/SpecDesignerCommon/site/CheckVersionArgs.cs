using System;

namespace SpecDesignerCommon.Site
{
	// Token: 0x0200009E RID: 158
	public class CheckVersionArgs : EventArgs
	{
		// Token: 0x170001CF RID: 463
		// (get) Token: 0x0600065E RID: 1630 RVA: 0x0001C907 File Offset: 0x0001AB07
		// (set) Token: 0x0600065F RID: 1631 RVA: 0x0001C90F File Offset: 0x0001AB0F
		public bool HasError { get; private set; }

		// Token: 0x170001D0 RID: 464
		// (get) Token: 0x06000660 RID: 1632 RVA: 0x0001C918 File Offset: 0x0001AB18
		// (set) Token: 0x06000661 RID: 1633 RVA: 0x0001C920 File Offset: 0x0001AB20
		public Exception ExceptionMessage { get; private set; }

		// Token: 0x06000662 RID: 1634 RVA: 0x0001C929 File Offset: 0x0001AB29
		public CheckVersionArgs(bool hasError, Exception exception)
			: this(hasError)
		{
			this.HasError = true;
			this.ExceptionMessage = exception;
		}

		// Token: 0x06000663 RID: 1635 RVA: 0x0001C940 File Offset: 0x0001AB40
		public CheckVersionArgs(bool hasError)
		{
			this.HasError = hasError;
			this.ExceptionMessage = null;
		}
	}
}
