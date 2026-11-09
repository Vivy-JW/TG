using System;

namespace SpecDesignerCommon.Exceptions
{
	// Token: 0x0200009F RID: 159
	public class NotInCurrentWorkspaceException : Exception
	{
		// Token: 0x06000664 RID: 1636 RVA: 0x0001C956 File Offset: 0x0001AB56
		public NotInCurrentWorkspaceException(string msg)
			: base(msg)
		{
		}
	}
}
