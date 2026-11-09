using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.CodeEditWindow.Helper
{
	// Token: 0x02000045 RID: 69
	public class FunctionGenerator
	{
		// Token: 0x060002F4 RID: 756 RVA: 0x000199A4 File Offset: 0x00017BA4
		public static IEnumerable<string> Generate(FglSpecification specification)
		{
			foreach (FglComponent com in specification.Components)
			{
				if ((com.Status & CodeSpecStatus.DELETE) != CodeSpecStatus.DELETE)
				{
					StringBuilder function = new StringBuilder();
					function.AppendLine("################################################################################");
					function.AppendLine(string.Format("# Descriptions...: {0}", com.Description));
					function.AppendLine("# Memo...........");
					function.AppendLine(string.Format("# Usage..........: CALL {0}", com.FunctionName));
					StringBuilder parameter = new StringBuilder();
					int index = 0;
					com.Returns.ToList<FglParameter>().ForEach(delegate(FglParameter r)
					{
						if (index > 0)
						{
							parameter.Append(", ");
						}
						parameter.Append(r.Name);
						index++;
					});
					if (parameter.Length > 0)
					{
						function.AppendLine(string.Format("#                  RETURNING {0}", parameter.ToString()));
					}
					index = 0;
					com.Inputs.ToList<FglParameter>().ForEach(delegate(FglParameter i)
					{
						if (index == 0)
						{
							function.AppendLine(string.Format("# Input parameter: {0}", i.NameWithPurpose));
						}
						else
						{
							function.AppendLine(string.Format("#                : {0}", i.NameWithPurpose));
						}
						index++;
					});
					index = 0;
					com.Returns.ToList<FglParameter>().ForEach(delegate(FglParameter i)
					{
						if (index == 0)
						{
							function.AppendLine(string.Format("# Return code....: {0}", i.NameWithPurpose));
						}
						else
						{
							function.AppendLine(string.Format("#                : {0}", i.NameWithPurpose));
						}
						index++;
					});
					function.AppendLine(string.Format("# Date & Author..: {0} By WHO?", DateTime.Now.ToShortDateString()));
					function.AppendLine("# Modify.........:");
					function.AppendLine("################################################################################");
					function.Append("PUBLIC FUNCTION ");
					StringBuilder parameters = new StringBuilder();
					if (com.Inputs.Count > 0)
					{
						com.Inputs.ToList<FglParameter>().ForEach(delegate(FglParameter i)
						{
							if (parameters.Length > 0)
							{
								parameters.Append(" ,");
							}
							parameters.Append(i.Name);
						});
						function.AppendLine(string.Format("{0}({1})", com.ID, parameters));
						parameters.Clear();
						com.Inputs.ToList<FglParameter>().ForEach(delegate(FglParameter i)
						{
							parameters.AppendLine(string.Format("   DEFINE {0} {1}", i.Name, i.ReferenceType));
						});
						function.AppendLine(parameters.ToString());
					}
					else
					{
						function.AppendLine(string.Format("{0}()", com.ID));
					}
					if (com.Returns.Count > 0)
					{
						parameters.Clear();
						com.Returns.ToList<FglParameter>().ForEach(delegate(FglParameter r)
						{
							parameters.AppendLine(string.Format("   DEFINE {0} {1}", r.Name, r.ReferenceType));
						});
						function.AppendLine(parameters.ToString());
						parameters.Clear();
						com.Returns.ToList<FglParameter>().ForEach(delegate(FglParameter r)
						{
							if (parameters.Length > 0)
							{
								parameters.Append(" ,");
							}
							parameters.Append(r.Name);
						});
						function.AppendLine(string.Format("   RETURN {0}", parameters));
					}
					function.Append("END FUNCTION");
					yield return function.ToString();
				}
			}
			yield break;
		}

		// Token: 0x060002F5 RID: 757 RVA: 0x00019A20 File Offset: 0x00017C20
		private static string GetDefinitionString(FglComponent com)
		{
			StringBuilder stringBuilder = new StringBuilder();
			if (com.Returns.Count > 0)
			{
				StringBuilder parameters = new StringBuilder();
				parameters.Clear();
				com.Returns.ToList<FglParameter>().ForEach(delegate(FglParameter r)
				{
					parameters.AppendLine(string.Format("  DEFINE {0} {1}", r.Name, r.Type));
				});
				stringBuilder.AppendLine(parameters.ToString());
				parameters.Clear();
				com.Returns.ToList<FglParameter>().ForEach(delegate(FglParameter r)
				{
					if (parameters.Length > 0)
					{
						parameters.Append(" ,");
					}
					parameters.Append(r.Name);
				});
				stringBuilder.AppendLine(string.Format("  RETURN {0}", parameters));
			}
			return stringBuilder.ToString();
		}
	}
}
