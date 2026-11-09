using System;

namespace SpecDesignerCommon.ViewModel
{
	// Token: 0x020000FD RID: 253
	public class CodeSpecStatusToString
	{
		// Token: 0x0600083D RID: 2109 RVA: 0x000241D9 File Offset: 0x000223D9
		public static string ToString(CodeSpecStatus status)
		{
			if ((status & CodeSpecStatus.DELETE) == CodeSpecStatus.DELETE)
			{
				return "d";
			}
			if ((status & CodeSpecStatus.CREATE) == CodeSpecStatus.CREATE)
			{
				return "u";
			}
			if ((status & CodeSpecStatus.MODIFY) == CodeSpecStatus.MODIFY)
			{
				return "u";
			}
			return "";
		}
	}
}
